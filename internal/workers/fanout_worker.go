package workers

import (
	"context"
	"sync"
	"time"

	"virast/internal/core/fanoutqueue"
	//"virast/internal/core/post"
	timelineEntity "virast/internal/core/timeline"
	//"virast/internal/core/user"
	fanoutPort "virast/internal/ports/fanoutqueue"
	followerPort "virast/internal/ports/follower"
	timelinePort "virast/internal/ports/timeline"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

type FanoutWorker struct {
	FanoutRepo   fanoutPort.FanoutRepository
	FanoutRedis  fanoutPort.FanoutRedis
	FollowerRepo followerPort.FollowerRepository
	TimelineRepo timelinePort.TimelineRepository
	BatchSize    int // تعداد رکوردهای batch برای Redis و timeline
	Concurrency  int // تعداد goroutine های همزمان
	Logger       *zap.Logger
}

func NewFanoutWorker(
	fanoutRepo fanoutPort.FanoutRepository,
	fanoutRedis fanoutPort.FanoutRedis,
	followerRepo followerPort.FollowerRepository,
	timelineRepo timelinePort.TimelineRepository,
	batchSize int,
	concurrency int,
	logger *zap.Logger,
) *FanoutWorker {
	if batchSize <= 0 {
		batchSize = 500
	}
	if concurrency <= 0 {
		concurrency = 8
	}
	return &FanoutWorker{
		FanoutRepo:   fanoutRepo,
		FanoutRedis:  fanoutRedis,
		FollowerRepo: followerRepo,
		TimelineRepo: timelineRepo,
		BatchSize:    batchSize,
		Concurrency:  concurrency,
		Logger:       logger,
	}
}

// Run گوش دادن به صف و توزیع پست‌ها
func (w *FanoutWorker) Run(ctx context.Context) {
	w.Logger.Info("🚀 FanoutWorker started", zap.Int("BatchSize", w.BatchSize), zap.Int("Concurrency", w.Concurrency))

	if w.Concurrency <= 0 {
		w.Concurrency = 8
	}

	jobs := make(chan *fanoutqueue.FanoutQueue, w.Concurrency*2)
	var wg sync.WaitGroup

	// راه‌اندازی ورکرها (ثابت)
	for i := 0; i < w.Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case fq, ok := <-jobs:
					if !ok {
						return // channel بسته شد → خروج تمیز
					}
					w.processFanout(ctx, fq)
				}
			}
		}(i)
	}

	// حلقه‌ی Producer: هر N میلی‌ثانیه pendingها را می‌گیرد و به صف می‌فرستد
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

producer:
	for {
		select {
		case <-ctx.Done():
			break producer
		case <-ticker.C:
			pendingPosts, err := w.FanoutRepo.GetPendingPosts(ctx, int64(w.BatchSize))
			if err != nil {
				w.Logger.Error("❌ Error fetching pending posts:", zap.Error(err))
				continue
			}

			// تزریق کارها به صف
			for _, fq := range pendingPosts {
				select {
				case <-ctx.Done():
					break producer
				case jobs <- fq:
				}
			}
		}
	}

	// خاموشی تمیز
	close(jobs) // به ورکرها بگو کار جدیدی نمیاد
	wg.Wait()   // صبر کن همه ورکرها تموم کنند
	w.Logger.Info("🛑 Fanout worker stopped")
	w.Logger.Info("✅ All fanout jobs processed")
}

// پردازش یک رکورد FanoutQueue
func (w *FanoutWorker) processFanout(ctx context.Context, fq *fanoutqueue.FanoutQueue) {
	if fq == nil || fq.PostID == uuid.Nil || fq.UserID == uuid.Nil {
		w.Logger.Error("❌ Invalid FanoutQueue record:", zap.Any("record", fq))
		return
	}

	w.Logger.Info("➡ Processing FanoutQueue", zap.String("PostID", fq.PostID.String()), zap.String("AuthorID", fq.UserID.String()))

	// گرفتن followers
	followers, err := w.FollowerRepo.GetFollowersByUserID(ctx, fq.UserID.String())
	if err != nil {
		w.Logger.Error("❌ Error fetching followers:", zap.Error(err))
		return
	}

	w.Logger.Info("👥 Found followers for user", zap.String("UserID", fq.UserID.String()), zap.Int("Count", len(followers)))

	if len(followers) == 0 {
		w.Logger.Warn("⚠️ No followers for user:", zap.String("UserID", fq.UserID.String()))
		if err := w.FanoutRepo.MarkDone(ctx, fq.ID); err != nil {
			w.Logger.Warn("⚠️ Warning: could not mark fanout_queue done:", zap.Error(err))
		}
		return
	}

	// تبدیل followers به []string
	var followerIDs []string
	for _, f := range followers {
		followerIDs = append(followerIDs, f.FollowerID.String())
	}

	// پردازش batch برای Redis ZSET
	for i := 0; i < len(followerIDs); i += w.BatchSize {
		end := min(i+w.BatchSize, len(followerIDs))
		batch := followerIDs[i:end]

		w.Logger.Info("📦 Processing batch", zap.Int("Count", len(batch)), zap.Int("From", i), zap.Int("To", end))

		// ZADD
		if err := w.FanoutRedis.PushPostToFollowers(ctx, fq.PostID.String(), batch); err != nil {
			w.Logger.Error("❌ Error pushing batch to ZSET:", zap.Error(err))
		} else {
			w.Logger.Info("✅ Pushed post to ZSET", zap.String("PostID", fq.PostID.String()), zap.Int("Count", len(batch)))
		}

		if len(batch) <= 0 {
			continue
		}

		// ساخت رکورد timeline به صورت batch
		addTimelines(ctx, w, fq, batch)
	}

	// بروزرسانی وضعیت رکورد fanout_queue به done
	if err := w.FanoutRepo.MarkDone(ctx, fq.ID); err != nil {
		w.Logger.Warn("⚠️ Warning: could not mark fanout_queue done:", zap.Error(err))
	} else {
		w.Logger.Info("✅ FanoutQueue record marked as done", zap.String("ID", fq.ID.String()))
	}
}

func addTimelines(ctx context.Context, w *FanoutWorker, fq *fanoutqueue.FanoutQueue, batch []string) {
	var timelines []*timelineEntity.Timeline
	for _, fid := range batch {
		timelines = append(timelines, &timelineEntity.Timeline{
			ID:     uuid.Must(uuid.NewV4()),
			UserID: uuid.FromStringOrNil(fid),
			PostID: fq.PostID,
			// CreatedAt: fq.CreatedAt,
			// DeletedAt: nil,
			// User:   user.User{},
			// Post:   post.Post{},
		})
	}

	w.Logger.Info("📝 Adding batch to timeline", zap.Int("Count", len(timelines)))
	if err := w.TimelineRepo.AddBatch(ctx, timelines); err != nil {
		w.Logger.Warn("⚠️ Warning: could not add batch to timeline", zap.Error(err))
	} else {
		w.Logger.Info("✅ Added timeline records for post", zap.String("PostID", fq.PostID.String()), zap.Int("Count", len(timelines)))
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

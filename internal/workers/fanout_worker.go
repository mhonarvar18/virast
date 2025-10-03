package workers

import (
	"context"
	"log"
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
)

type FanoutWorker struct {
	FanoutRepo   fanoutPort.FanoutRepository
	FanoutRedis  fanoutPort.FanoutRedis
	FollowerRepo followerPort.FollowerRepository
	TimelineRepo timelinePort.TimelineRepository
	BatchSize    int // تعداد رکوردهای batch برای Redis و timeline
	Concurrency  int // تعداد goroutine های همزمان
}

func NewFanoutWorker(
	fanoutRepo fanoutPort.FanoutRepository,
	fanoutRedis fanoutPort.FanoutRedis,
	followerRepo followerPort.FollowerRepository,
	timelineRepo timelinePort.TimelineRepository,
	batchSize int,
	concurrency int,
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
	}
}

// Run گوش دادن به صف و توزیع پست‌ها
func (w *FanoutWorker) Run(ctx context.Context) {
	log.Println("🚀 FanoutWorker started")

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
				log.Println("❌ Error fetching pending posts:", err)
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
	log.Println("🛑 Fanout worker stopped")
	log.Println("✅ All fanout jobs processed")
}

// پردازش یک رکورد FanoutQueue
func (w *FanoutWorker) processFanout(ctx context.Context, fq *fanoutqueue.FanoutQueue) {
	if fq == nil || fq.PostID == uuid.Nil || fq.UserID == uuid.Nil {
		log.Println("❌ Invalid FanoutQueue record:", fq)
		return
	}

	log.Printf("➡ Processing FanoutQueue: PostID=%s AuthorID=%s\n", fq.PostID, fq.UserID)

	// گرفتن followers
	followers, err := w.FollowerRepo.GetFollowersByUserID(ctx, fq.UserID.String())
	if err != nil {
		log.Println("❌ Error fetching followers:", err)
		return
	}

	log.Printf("👥 Found %d followers for user %s\n", len(followers), fq.UserID)

	if len(followers) == 0 {
		log.Println("⚠️ No followers for user:", fq.UserID)
		if err := w.FanoutRepo.MarkDone(ctx, fq.ID); err != nil {
			log.Println("⚠️ Warning: could not mark fanout_queue done:", err)
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

		log.Printf("📦 Processing batch: %d followers (from %d to %d)\n", len(batch), i, end)

		// ZADD
		if err := w.FanoutRedis.PushPostToFollowers(ctx, fq.PostID.String(), batch); err != nil {
			log.Println("❌ Error pushing batch to ZSET:", err)
		} else {
			log.Printf("✅ Pushed post %s to ZSET for %d followers\n", fq.PostID, len(batch))
		}

		if len(batch) <= 0 {
			continue
		}

		// ساخت رکورد timeline به صورت batch
		addTimelines(ctx, w, fq, batch)
	}

	// بروزرسانی وضعیت رکورد fanout_queue به done
	if err := w.FanoutRepo.MarkDone(ctx, fq.ID); err != nil {
		log.Println("⚠️ Warning: could not mark fanout_queue done:", err)
	} else {
		log.Printf("✅ FanoutQueue record marked as done: %s\n", fq.ID)
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

	log.Printf("📝 Adding batch to timeline: %d records\n", len(timelines))
	if err := w.TimelineRepo.AddBatch(ctx, timelines); err != nil {
		log.Println("⚠️ Warning: could not add batch to timeline:", err)
	} else {
		log.Printf("✅ Added %d timeline records for post %s\n", len(timelines), fq.PostID)
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

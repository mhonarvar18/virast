package postapp

import (
	"context"
	"fmt"

	//fanoutQueueEntity "virast/internal/core/fanoutqueue"
	"virast/internal/config"
	"virast/internal/core/fanoutqueue"
	postEntity "virast/internal/core/post"

	//"virast/internal/core/timeline"
	fanoutPort "virast/internal/ports/fanoutqueue"
	followerPort "virast/internal/ports/follower"
	postPort "virast/internal/ports/post"
	timelinePort "virast/internal/ports/timeline"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

type PostService struct {
	PostRepository     postPort.PostRepository
	FanoutRepository   fanoutPort.FanoutRepository     // تزریق شده
	FanoutRedis        fanoutPort.FanoutRedis          // تزریق شده
	FollowerRepository followerPort.FollowerRepository // برای گرفتن followers
	TimelineRepository timelinePort.TimelineRepository // برای ذخیره در جدول timeline
}

func NewPostService(
	postRepo postPort.PostRepository,
	fanoutRepo fanoutPort.FanoutRepository,
	fanoutRedis fanoutPort.FanoutRedis,
	followerRepo followerPort.FollowerRepository,
	timelineRepo timelinePort.TimelineRepository,
) *PostService {
	return &PostService{
		FollowerRepository: followerRepo,
		FanoutRepository:   fanoutRepo,
		FanoutRedis:        fanoutRedis,
		PostRepository:     postRepo,
		TimelineRepository: timelineRepo,
	}
}

// CreatePost ایجاد یک پست جدید و اضافه کردن به FanoutQueue
func (s *PostService) CreatePost(ctx context.Context, content, userID string) (*postPort.PostDTO, error) {
	config.Logger.Info("🚀 CreatePost called", zap.String("userID", userID), zap.String("content", content))

	// اعتبارسنجی UUID
	uid, err := uuid.FromString(userID)
	if err != nil {
		config.Logger.Error("❌ Invalid userID", zap.String("userID", userID), zap.Error(err))
		return nil, fmt.Errorf("invalid userID: %w", err)
	}

	// 1️⃣ ایجاد رکورد Post
	post := &postEntity.Post{
		ID:      uuid.Must(uuid.NewV4()),
		Content: content,
		UserID:  uid,
	}

	createdPost, err := s.PostRepository.Create(post)
	if err != nil {
		config.Logger.Error("❌ Failed to create post", zap.String("userID", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to create post: %w", err)
	}
	config.Logger.Info("✅ Post created", zap.String("postID", createdPost.ID.String()), zap.String("userID", createdPost.UserID.String()))

	// 2️⃣ ایجاد رکورد FanoutQueue (pending)
	fq := &fanoutqueue.FanoutQueue{
		ID:     uuid.Must(uuid.NewV4()),
		PostID: createdPost.ID,
		UserID: createdPost.UserID,
		Status: "pending",
	}

	fanoutRecord, err := s.FanoutRepository.Create(ctx, fq)
	if err != nil {
		config.Logger.Error("❌ Could not add to fanout_queue", zap.Error(err))
	} else {
		config.Logger.Info("✅ Added to fanout_queue", zap.String("fanoutID", fanoutRecord.ID.String()), zap.String("postID", fanoutRecord.PostID.String()))
	}

	// 3️⃣ پیام برای FanoutWorker (برای ZSET)
	if err := s.FanoutRedis.PushPostToFollowers(ctx, createdPost.ID.String(), []string{createdPost.UserID.String()}); err != nil {
		config.Logger.Error("❌ Could not push post to Redis ZSET", zap.Error(err))
	} else {
		config.Logger.Info("✅ Post pushed to Redis ZSET for user", zap.String("userID", createdPost.UserID.String()))
	}

	config.Logger.Info("🚀 CreatePost completed", zap.String("postID", createdPost.ID.String()))
	return &postPort.PostDTO{
		ID:      createdPost.ID.String(),
		Content: createdPost.Content,
	}, nil
}

package postapp

import (
	"context"
	"fmt"

	//fanoutQueueEntity "virast/internal/core/fanoutqueue"
	"virast/internal/core/fanoutqueue"
	postEntity "virast/internal/core/post"

	//"virast/internal/core/timeline"
	fanoutPort "virast/internal/ports/fanoutqueue"
	followerPort "virast/internal/ports/follower"
	postPort "virast/internal/ports/post"
	timelinePort "virast/internal/ports/timeline"

	"github.com/gofrs/uuid"
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
	fmt.Println("🚀 CreatePost called with userID:", userID, "content:", content)

	// اعتبارسنجی UUID
	uid, err := uuid.FromString(userID)
	if err != nil {
		fmt.Println("❌ Invalid userID:", userID, "error:", err)
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
		fmt.Println("❌ Failed to create post for userID:", userID, "error:", err)
		return nil, fmt.Errorf("failed to create post: %w", err)
	}
	fmt.Println("✅ Created post:", createdPost.ID, "for user:", createdPost.UserID)

	// 2️⃣ ایجاد رکورد FanoutQueue (pending)
	fq := &fanoutqueue.FanoutQueue{
		ID:     uuid.Must(uuid.NewV4()),
		PostID: createdPost.ID,
		UserID: createdPost.UserID,
		Status: "pending",
	}

	fanoutRecord, err := s.FanoutRepository.Create(ctx, fq)
	if err != nil {
		fmt.Println("⚠️ Warning: could not add to fanout_queue:", err)
	} else {
		fmt.Println("✅ FanoutQueue record created:", fanoutRecord.ID)
	}

	// 3️⃣ پیام برای FanoutWorker (برای ZSET)
	if err := s.FanoutRedis.PushPostToFollowers(ctx, createdPost.ID.String(), []string{createdPost.UserID.String()}); err != nil {
		fmt.Println("⚠️ Warning: could not push post to Redis ZSET:", err)
	} else {
		fmt.Println("✅ Post pushed to Redis ZSET for user:", createdPost.UserID)
	}

	fmt.Println("🚀 CreatePost completed for postID:", createdPost.ID)
	return &postPort.PostDTO{
		ID:      createdPost.ID.String(),
		Content: createdPost.Content,
	}, nil
}

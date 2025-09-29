package database

import (
	"context"
	"fmt"
	"time"
	"virast/internal/config"
	postEntity "virast/internal/core/post"
	timelineEntity "virast/internal/core/timeline"
	postPort "virast/internal/ports/post"
	userPort "virast/internal/ports/user"

	"github.com/gofrs/uuid"
)

type TimelineRepositoryDatabase struct{}

func NewtimelineRepositoryDatabase() *TimelineRepositoryDatabase {
	return &TimelineRepositoryDatabase{}
}

// Add اضافه کردن یک پست به جدول timeline
func (repo *TimelineRepositoryDatabase) Add(ctx context.Context, tl *timelineEntity.Timeline) error {
	if tl.ID == uuid.Nil {
		tl.ID = uuid.Must(uuid.NewV4())
	}
	if tl.CreatedAt.IsZero() {
		tl.CreatedAt = time.Now()
	}

	if err := config.DB.Create(tl).Error; err != nil {
		fmt.Println("Error adding to timeline:", err)
		return err
	}
	return nil
}

// AddBatch اضافه کردن چندین پست به جدول timeline به صورت دسته‌ای
func (repo *TimelineRepositoryDatabase) AddBatch(ctx context.Context, timelines []*timelineEntity.Timeline) error {
	if len(timelines) == 0 {
		fmt.Println("⚠️ No timelines to add")
		return nil
	}

	// چک کنیم هیچ pointer nil نباشه
	for i, tl := range timelines {
		if tl == nil {
			return fmt.Errorf("timeline[%d] is nil", i)
		}
		if tl.ID == uuid.Nil {
			tl.ID = uuid.Must(uuid.NewV4())
		}
		if tl.UserID == uuid.Nil || tl.PostID == uuid.Nil {
			return fmt.Errorf("timeline[%d] has nil UserID or PostID", i)
		}
		if tl.CreatedAt.IsZero() {
			tl.CreatedAt = time.Now()
		}
		fmt.Printf("Timeline[%d] ready: ID=%s UserID=%s PostID=%s\n", i, tl.ID, tl.UserID, tl.PostID)
	}

	// insert batch
	if err := config.DB.CreateInBatches(&timelines, len(timelines)).Error; err != nil {
		return fmt.Errorf("error adding batch to timeline: %w", err)
	}

	fmt.Printf("✅ Added batch of %d timelines\n", len(timelines))
	return nil
}

// GetTimelineByUserID بازیابی تایم‌لاین کاربر با start و limit
func (repo *TimelineRepositoryDatabase) GetTimelineByUserID(ctx context.Context, userID string, start, limit int64) ([]*postPort.PostDTO, error) {
	key := "timeline:" + userID

	// 1️⃣ گرفتن postIDها از Redis ZSET
	postIDs, err := config.RedisClient.ZRevRange(ctx, key, start, start+limit-1).Result()
	if err != nil {
		return nil, err
	}

	posts := make([]*postPort.PostDTO, 0, len(postIDs))

	// 2️⃣ برای هر postID دیتای کامل post + user از دیتابیس
	for _, pid := range postIDs {
		var postEntity postEntity.Post
		if err := config.DB.Preload("User").First(&postEntity, "id = ?", pid).Error; err != nil {
			fmt.Println("Warning: post not found:", pid)
			continue
		}

		posts = append(posts, &postPort.PostDTO{
			ID:      postEntity.ID.String(),
			Content: postEntity.Content,
			UserID:  postEntity.UserID.String(),
			User: &userPort.UserDTO{
				ID:       postEntity.User.ID.String(),
				Username: postEntity.User.Username,
				Mobile:   postEntity.User.Mobile,
			},
			CreatedAt: postEntity.CreatedAt.String(),
		})
	}

	return posts, nil
}

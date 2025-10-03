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
	"go.uber.org/zap"
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
		config.Logger.Error("Error adding to timeline:", zap.Error(err))
		return err
	}

	config.Logger.Info("Added timeline record",
		zap.String("timelineID", tl.ID.String()),
		zap.String("userID", tl.UserID.String()),
		zap.String("postID", tl.PostID.String()),
	)
	return nil
}

// AddBatch اضافه کردن چندین پست به جدول timeline به صورت دسته‌ای
func (repo *TimelineRepositoryDatabase) AddBatch(ctx context.Context, timelines []*timelineEntity.Timeline) error {
	if len(timelines) == 0 {
		config.Logger.Warn("⚠️ No timelines to add")
		return nil
	}

	// چک کنیم هیچ pointer nil نباشه
	for i, tl := range timelines {
		if tl == nil {
			config.Logger.Error("Timeline is nil", zap.Int("index", i))
			return fmt.Errorf("timeline[%d] is nil", i)
		}
		if tl.ID == uuid.Nil {
			tl.ID = uuid.Must(uuid.NewV4())
		}
		if tl.UserID == uuid.Nil || tl.PostID == uuid.Nil {
			config.Logger.Error("Timeline has nil UserID or PostID", zap.Int("index", i))
			return fmt.Errorf("timeline[%d] has nil UserID or PostID", i)
		}
		if tl.CreatedAt.IsZero() {
			tl.CreatedAt = time.Now()
		}
		config.Logger.Info("Timeline ready",
			zap.Int("index", i),
			zap.String("ID", tl.ID.String()),
			zap.String("UserID", tl.UserID.String()),
			zap.String("PostID", tl.PostID.String()),
		)
	}

	// insert batch
	if err := config.DB.CreateInBatches(&timelines, len(timelines)).Error; err != nil {
		config.Logger.Error("error adding batch to timeline", zap.Error(err))
		return err
	}

	config.Logger.Info("✅ Added batch of timelines", zap.Int("count", len(timelines)))
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
			config.Logger.Warn("Warning: post not found:", zap.String("postID", pid), zap.Error(err))
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

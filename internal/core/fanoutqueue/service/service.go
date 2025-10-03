package fanoutqueueapp

import (
	"context"
	"time"
	"virast/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type FanoutService struct{}

func NewFanoutService() *FanoutService {
	return &FanoutService{}
}

// PushPostToFollowers تایم‌لاین تمام دنبال‌کننده‌ها را بروزرسانی می‌کند
func (s *FanoutService) PushPostToFollowers(postID, userID string, followers []string) error {
	config.Logger.Info("Pushing post", zap.String("postID", postID), zap.Strings("followerIDs", followers))
	ctx := context.Background()
	for _, followerID := range followers {
		key := "timeline:" + followerID

		config.Logger.Info("Adding post to timeline", zap.String("postID", postID), zap.String("timelineKey", key))

		// member برای ZAdd
		z := &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: postID,
		}

		// اضافه کردن به ZSET
		if err := config.RedisClient.ZAdd(ctx, key, z).Err(); err != nil {
			return err
		}

		config.Logger.Info("Added post to timeline", zap.String("postID", postID), zap.String("timelineKey", key))

		// بررسی محتویات ZSET بعد از افزودن
		config.Logger.Info("Redis address:", zap.String("address", config.RedisClient.Options().Addr))
		posts, err := config.RedisClient.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			config.Logger.Error("Error reading ZSET:", zap.Error(err))
			continue
		}
		config.Logger.Info("Timeline contents for "+followerID, zap.Any("posts", posts))
		for _, p := range posts {
			config.Logger.Info("Timeline post", zap.String("postID", p.Member.(string)), zap.Float64("score", p.Score))
		}
	}
	return nil
}

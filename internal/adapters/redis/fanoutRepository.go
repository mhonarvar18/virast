package redis

import (
	"context"
	"time"
	"virast/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type FanoutRepositoryRedis struct {
	Client *redis.Client
}

func NewFanoutRepositoryRedis(client *redis.Client) *FanoutRepositoryRedis {
	return &FanoutRepositoryRedis{
		Client: client,
	}
}

// PushPostToFollowers: اضافه کردن postID به timeline ZSET تمام followers
func (r *FanoutRepositoryRedis) PushPostToFollowers(ctx context.Context, postID string, followerIDs []string) error {
	config.Logger.Info("!!!!!!!!!!!!!!!!!!!!")
	config.Logger.Info("Pushing post", zap.String("postID", postID), zap.Strings("followerIDs", followerIDs))
	config.Logger.Info("!!!!!!!!!!!!!!!!!!!!!!!!!")
	for _, followerID := range followerIDs {
		key := "timeline:" + followerID

		z := &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: postID,
		}

		config.Logger.Info("Adding to ZSET", zap.String("key", key), zap.String("postID", postID))

		if err := r.Client.ZAdd(ctx, key, z).Err(); err != nil {
			return err
		}

		// optional: چاپ برای debug
		config.Logger.Info("Added post to timeline", zap.String("postID", postID), zap.String("timelineKey", key))
	}

	return nil
}

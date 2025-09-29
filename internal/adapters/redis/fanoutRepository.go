package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
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
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Println("Pushing post", postID, "to followers:", followerIDs)
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!")
	for _, followerID := range followerIDs {
		key := "timeline:" + followerID

		z := &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: postID,
		}

		fmt.Println("Adding to ZSET", key, "postID:", postID)

		if err := r.Client.ZAdd(ctx, key, z).Err(); err != nil {
			return err
		}

		// optional: چاپ برای debug
		fmt.Println("Added post", postID, "to", key)
	}

	return nil
}
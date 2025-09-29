package fanoutqueueapp

import (
	"context"
	"fmt"
	"time"
	"virast/internal/config"

	"github.com/go-redis/redis/v8"
)

type FanoutService struct{}

func NewFanoutService() *FanoutService {
	return &FanoutService{}
}

// PushPostToFollowers تایم‌لاین تمام دنبال‌کننده‌ها را بروزرسانی می‌کند
func (s *FanoutService) PushPostToFollowers(postID, userID string, followers []string) error {
	fmt.Println("Pushing post", postID, "to followers:", followers)
	ctx := context.Background()
	for _, followerID := range followers {
		key := "timeline:" + followerID

		fmt.Println("Adding post", postID, "to timeline key:", key)

		// member برای ZAdd
		z := &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: postID,
		}

		// اضافه کردن به ZSET
		if err := config.RedisClient.ZAdd(ctx, key, z).Err(); err != nil {
			return err
		}

		fmt.Println("Added post", postID, "to timeline of follower", followerID)

		// بررسی محتویات ZSET بعد از افزودن
		fmt.Println("Redis address:", config.RedisClient.Options().Addr)
		posts, err := config.RedisClient.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			fmt.Println("Error reading ZSET:", err)
			continue
		}
		fmt.Println("Timeline contents for", followerID, ":")
		for _, p := range posts {
			fmt.Printf("  PostID: %v, Score: %v\n", p.Member, p.Score)
		}
	}
	return nil
}

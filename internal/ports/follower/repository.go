package follower

import (
	"context"
	"virast/internal/core/follower"
)

// FollowerRepository پورت برای ذخیره‌سازی و بازیابی دنبال‌کنندگان
type FollowerRepository interface {
	FollowUser(ctx context.Context, follower *follower.Follower) (*follower.Follower, error)
	UnfollowUser(ctx context.Context, followerID, followeeID string) error
	GetFollowersByUserID(ctx context.Context, userID string) ([]*follower.Follower, error)
	GetFollowingByUserID(ctx context.Context, followerID string) ([]*follower.Follower, error)
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
}

// DTOها برای UseCase
type FollowerDTO struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	FollowerID string `json:"followerId"`
}

type UnfollowDTO struct {
	FollowerID string `json:"followerId"`
}

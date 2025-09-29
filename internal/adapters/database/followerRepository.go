package database

import (
	"context"
	"virast/internal/config"
	"virast/internal/core/follower"
)

// FollowerRepositoryDatabase پیاده‌سازی FollowerRepository برای دیتابیس
type FollowerRepositoryDatabase struct{}

// NewFollowerRepositoryDatabase سازنده FollowerRepositoryDatabase
func NewFollowerRepositoryDatabase() *FollowerRepositoryDatabase {
	return &FollowerRepositoryDatabase{}
}

func (repo *FollowerRepositoryDatabase) FollowUser(ctx context.Context, follower *follower.Follower) (*follower.Follower, error) {
	if err := config.DB.Create(follower).Error; err != nil {
		return nil, err
	}
	return follower, nil
}

func (repo *FollowerRepositoryDatabase) UnfollowUser(ctx context.Context, followerID, followeeID string) error {
	if err := config.DB.Where("follower_id = ? AND user_id = ?", followerID, followeeID).Delete(&follower.Follower{}).Error; err != nil {
		return err
	}
	return nil
}

func (repo *FollowerRepositoryDatabase) GetFollowersByUserID(ctx context.Context, userID string) ([]*follower.Follower, error) {
	var followers []*follower.Follower
	if err := config.DB.Where("user_id = ?", userID).Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func (repo *FollowerRepositoryDatabase) GetFollowingByUserID(ctx context.Context, followerID string) ([]*follower.Follower, error) {
	var following []*follower.Follower
	if err := config.DB.Where("follower_id = ?", followerID).Find(&following).Error; err != nil {
		return nil, err
	}
	return following, nil
}

func (repo *FollowerRepositoryDatabase) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	var count int64
	if err := config.DB.Model(&follower.Follower{}).Where("follower_id = ? AND user_id = ?", followerID, followeeID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

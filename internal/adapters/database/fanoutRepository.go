package database

import (
	"context"
	"github.com/gofrs/uuid"
	"virast/internal/config"
	"virast/internal/core/fanoutqueue"
)

type FanoutRepositoryDatabase struct{}

func NewFanoutRepositoryDatabase() *FanoutRepositoryDatabase {
	return &FanoutRepositoryDatabase{}
}

func (repo *FanoutRepositoryDatabase) Create(ctx context.Context, fanout *fanoutqueue.FanoutQueue) (*fanoutqueue.FanoutQueue, error) {
	if err := config.DB.Create(fanout).Error; err != nil {
		return nil, err
	}
	return fanout, nil
}

func (repo *FanoutRepositoryDatabase) GetPendingPosts(ctx context.Context, limit int64) ([]*fanoutqueue.FanoutQueue, error) {
	var fanouts []*fanoutqueue.FanoutQueue
	if err := config.DB.
		//Preload("Post"). // بارگذاری relation با Post
		//Preload("User"). // بارگذاری relation با User
		Where("status = ?", "pending").
		Limit(int(limit)).
		Find(&fanouts).Error; err != nil {
		return nil, err
	}
	return fanouts, nil
}

func (repo *FanoutRepositoryDatabase) MarkDone(ctx context.Context, id uuid.UUID) error {
	if err := config.DB.Model(&fanoutqueue.FanoutQueue{}).
		Where("id = ?", id).
		Update("status", "done").Error; err != nil {
		return err
	}
	return nil
}

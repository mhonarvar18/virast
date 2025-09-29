package fanout

import (
	"context"
	"virast/internal/core/fanoutqueue"

	"github.com/gofrs/uuid"
)

type FanoutRepository interface {
	Create(ctx context.Context, fanout *fanoutqueue.FanoutQueue) (*fanoutqueue.FanoutQueue, error)
	GetPendingPosts(ctx context.Context, limit int64) ([]*fanoutqueue.FanoutQueue, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
}

type FanoutRedis interface {
	PushPostToFollowers(ctx context.Context, postID string, followerIDs []string) error
}

// مدل پیام در صف
type FanoutMessage struct {
	PostID   string
	AuthorID string
}

type FanoutQueueDTO struct {
	ID     uuid.UUID
	PostID uuid.UUID
	UserID uuid.UUID
	Status string // pending, done, failed
}

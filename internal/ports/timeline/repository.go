package timeline

import (
	"context"
	"virast/internal/core/timeline"
	postPort "virast/internal/ports/post"
)
type TimelineRepository interface {
	GetTimelineByUserID(ctx context.Context, userID string, start, limit int64) ([]*postPort.PostDTO, error)
	Add(ctx context.Context, tl *timeline.Timeline) error
	AddBatch(ctx context.Context, timelines []*timeline.Timeline) error
}

package timelineapp

import (
	"context"
	"virast/internal/core/timeline"
	postPort "virast/internal/ports/post"
	timelinePort "virast/internal/ports/timeline"
)

type TimelineService struct {
	TimelineRepository timelinePort.TimelineRepository
}

func NewTimelineService(timelineRepo timelinePort.TimelineRepository) *TimelineService {
	return &TimelineService{
		TimelineRepository: timelineRepo,
	}
}

// GetTimelineByUserID دریافت تایم‌لاین یک کاربر با استفاده از شناسه کاربری، شروع و محدودیت
func (s *TimelineService) GetTimelineByUserID(ctx context.Context, userID string, start, limit int64) ([]*postPort.PostDTO, error) {
	return s.TimelineRepository.GetTimelineByUserID(ctx, userID, start, limit)
}

func (s *TimelineService) Add(ctx context.Context, tl *timeline.Timeline) error {
	return s.TimelineRepository.Add(ctx, tl)
}

package repository

import (
	"ai-video/internal/gen/model"
	"context"
)

type TrackingEventRepo struct{}

func NewTrackingEventRepo() *TrackingEventRepo {
	return &TrackingEventRepo{}
}
func (r *TrackingEventRepo) Create(ctx context.Context, item *model.VideoTrackingEvent) error {
	return qFrom(ctx).VideoTrackingEvent.WithContext(ctx).Create(item)
}

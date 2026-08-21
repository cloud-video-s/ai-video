package service

import (
	"ai-video/internal/gen/model"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/repository"
)

const (
	maxTrackingExtendedFields     = 16
	maxTrackingExtendedFieldsJSON = 4 << 10
)

var (
	ErrTrackingEventInvalid = errors.New("invalid tracking event")
	trackingFieldName       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)
)

type TrackingEventRequest struct {
	TrackingType  string `json:"tracking_type" binding:"required,max=64"`
	ExtensionType string `json:"extension_type"`
	ModelID       uint64 `json:"model_id"`
}

type TrackingEventClientContext struct {
	AppCode     string
	PackageCode string
	AppVersion  string
	ChannelCode string
	CountryCode string
	PhoneModel  string
	SystemType  int
}

type trackingEventWriter interface {
	Create(ctx context.Context, item *model.VideoTrackingEvent) error
}

type TrackingEventService struct {
	repo trackingEventWriter
	now  func() time.Time
}

func NewTrackingEventService() *TrackingEventService {
	return newTrackingEventService(repository.NewTrackingEventRepo())
}

func newTrackingEventService(repo trackingEventWriter) *TrackingEventService {
	return &TrackingEventService{repo: repo, now: time.Now}
}

// Report validates and appends one event. It deliberately has no deduplication:
// one client report must always produce one row for occurrence counting.
func (s *TrackingEventService) Report(ctx context.Context, userID uint64, client TrackingEventClientContext, req TrackingEventRequest) (*model.VideoTrackingEvent, error) {
	if userID == 0 {
		return nil, fmt.Errorf("%w: user_id is required", ErrTrackingEventInvalid)
	}
	trackingType, extensionType, err := normalizeTrackingEvent(req)
	if err != nil {
		return nil, err
	}

	systemType := uint8(0)
	if client.SystemType >= 0 && client.SystemType <= 255 {
		systemType = uint8(client.SystemType)
	}
	now := s.now()
	record := &model.VideoTrackingEvent{
		UserID:        userID,
		TrackingType:  string(trackingType),
		ExtensionType: extensionType,
		ModelID:       req.ModelID,
		AppCode:       strings.TrimSpace(client.AppCode),
		PackageCode:   strings.TrimSpace(client.PackageCode),
		AppVersion:    strings.TrimSpace(client.AppVersion),
		ChannelCode:   strings.TrimSpace(client.ChannelCode),
		CountryCode:   strings.TrimSpace(client.CountryCode),
		PhoneModel:    strings.TrimSpace(client.PhoneModel),
		SystemType:    systemType,
		CreatedAt:     now,
	}
	if err = s.repo.Create(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func normalizeTrackingEvent(req TrackingEventRequest) (domain.TrackingDataType, string, error) {
	dataType, ok := domain.ParseTrackingDataType(req.TrackingType)
	if !ok {
		return "", "", fmt.Errorf("%w: unsupported data_type", ErrTrackingEventInvalid)
	}

	if dataType.SupportsExtendedFields() && req.ExtensionType == "" {
		return "", "", fmt.Errorf("%w: extended fields is required", ErrTrackingEventInvalid)
	}
	return dataType, strings.TrimSpace(req.ExtensionType), nil
}

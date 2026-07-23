package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ai-video/internal/pkg/cache"
	"ai-video/internal/repository"
)

var ErrTemplateFavoriteBusy = errors.New("template favorite operation is already in progress")

const templateFavoriteLockTTL = 10 * time.Second

type TemplateFavoriteService struct {
	repo *repository.TemplateFavoriteRepo
}

func NewTemplateFavoriteService() *TemplateFavoriteService {
	return &TemplateFavoriteService{repo: repository.NewTemplateFavoriteRepo()}
}

type TemplateFavoriteResponse struct {
	TemplateID    uint64 `json:"template_id"`
	Favorited     bool   `json:"favorited"`
	FavoriteCount uint64 `json:"favorite_count"`
}

func (s *TemplateFavoriteService) Set(ctx context.Context, userID, templateID uint64, favorited bool) (*TemplateFavoriteResponse, error) {
	lockKey := fmt.Sprintf("lock:template-favorite:%d:%d", userID, templateID)
	token, acquired, err := cache.AcquireLock(lockKey, templateFavoriteLockTTL)
	if err != nil {
		return nil, fmt.Errorf("acquire template favorite lock: %w", err)
	}
	if !acquired {
		return nil, ErrTemplateFavoriteBusy
	}
	defer func() { _ = cache.ReleaseLock(lockKey, token) }()

	state, err := s.repo.SetFavorite(ctx, userID, templateID, favorited)
	if err != nil {
		return nil, err
	}
	return &TemplateFavoriteResponse{
		TemplateID: state.TemplateID, Favorited: state.Favorited, FavoriteCount: state.FavoriteCount,
	}, nil
}

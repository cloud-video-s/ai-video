package repository

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TemplateFavoriteRepo struct{}

func NewTemplateFavoriteRepo() *TemplateFavoriteRepo {
	return &TemplateFavoriteRepo{}
}

type TemplateFavoriteState struct {
	TemplateID    uint64
	Favorited     bool
	FavoriteCount uint64
}

// SetFavorite 在事务中更新收藏关系，并从关联表实时计算收藏数量。
func (r *TemplateFavoriteRepo) SetFavorite(ctx context.Context, userID, templateID uint64, favorited bool) (*TemplateFavoriteState, error) {
	state := &TemplateFavoriteState{TemplateID: templateID, Favorited: favorited}
	err := Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		template := q.VideoTemplate
		templateDAO := template.WithContext(txCtx).Select(template.ID).Where(template.ID.Eq(templateID))
		if favorited {
			templateDAO = templateDAO.Where(template.Status.Eq(1))
		}
		if _, err := templateDAO.First(); err != nil {
			return err
		}

		favorite := q.VideoUserTemplateFavorite
		favoriteDAO := favorite.WithContext(txCtx).Where(
			favorite.UserID.Eq(userID), favorite.TemplateID.Eq(templateID),
		)
		if favorited {
			if _, err := favoriteDAO.First(); errors.Is(err, gorm.ErrRecordNotFound) {
				if err := favorite.WithContext(txCtx).Create(&model.VideoUserTemplateFavorite{
					UserID: userID, TemplateID: templateID,
				}); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		} else {
			if _, err := favoriteDAO.Delete(); err != nil {
				return err
			}
		}
		count, err := favorite.WithContext(txCtx).Where(favorite.TemplateID.Eq(templateID)).Count()
		if err != nil {
			return err
		}
		state.FavoriteCount = uint64(count)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (r *TemplateFavoriteRepo) GetUserFavorite(ctx context.Context, userID, templateID uint64) bool {
	key := fmt.Sprintf("user_favorite_key_%d_%d", userID, templateID)
	val, _ := config.Redis.Get(ctx, key).Bool()
	if val {
		return true
	}
	q := qFrom(ctx).VideoUserTemplateFavorite
	count, _ := q.WithContext(ctx).Where(q.UserID.Eq(userID)).Where(q.TemplateID.Eq(templateID)).Count()
	if count > 0 {
		config.Redis.Set(ctx, key, "1", 60*time.Second)
		return true
	}
	return false
}

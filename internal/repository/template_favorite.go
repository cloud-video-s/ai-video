package repository

import (
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/cache"
	"context"
	"errors"
	"fmt"

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
		templateDAO := template.WithContext(txCtx).Where(template.ID.Eq(templateID))
		if favorited {
			templateDAO = templateDAO.Where(template.Status.Eq(1))
		}
		count, err := templateDAO.Count()
		if count == 0 || err != nil {
			return fmt.Errorf("template is empty")
		}
		favorite := q.VideoUserTemplateFavorite
		favoriteDAO := favorite.WithContext(txCtx).Where(
			favorite.UserID.Eq(userID), favorite.TemplateID.Eq(templateID),
		)
		if favorited {
			if _, err = favoriteDAO.First(); errors.Is(err, gorm.ErrRecordNotFound) {
				if err = favorite.WithContext(txCtx).Create(&model.VideoUserTemplateFavorite{
					UserID: userID, TemplateID: templateID,
				}); err != nil {
					return err
				}
				_, err = templateDAO.UpdateColumn(template.LikeCount, gorm.Expr("like_count + ?", 1))
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		} else {
			if _, err = favoriteDAO.Delete(); err != nil {
				return err
			}
			_, err = templateDAO.UpdateColumn(template.LikeCount, gorm.Expr("like_count - ?", 1))
			if err != nil {
				return err
			}
		}
		count, err = favorite.WithContext(txCtx).Where(favorite.TemplateID.Eq(templateID)).Count()
		if err != nil {
			return err
		}
		state.FavoriteCount = uint64(count)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = cache.SetTemplateFavorite(userID, templateID, state.Favorited); err != nil {
		return nil, fmt.Errorf("sync template favorite cache: %w", err)
	}
	return state, nil
}

func (r *TemplateFavoriteRepo) GetUserFavorite(ctx context.Context, userID, templateID uint64) bool {
	if favorited, found := cache.GetTemplateFavorite(userID, templateID); found {
		return favorited
	}

	q := qFrom(ctx).VideoUserTemplateFavorite
	count, err := q.WithContext(ctx).Where(q.UserID.Eq(userID)).Where(q.TemplateID.Eq(templateID)).Count()
	if err != nil {
		return false
	}

	favorited := count > 0
	_ = cache.SetTemplateFavorite(userID, templateID, favorited)
	return favorited
}

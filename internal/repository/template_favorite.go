package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// SetFavorite changes a user's favorite state transactionally. The unique
// user/template index makes favorite idempotent, and the guarded decrement
// prevents favorite_count from ever becoming negative.
func (r *TemplateFavoriteRepo) SetFavorite(ctx context.Context, userID, templateID uint64, favorited bool) (*TemplateFavoriteState, error) {
	state := &TemplateFavoriteState{TemplateID: templateID, Favorited: favorited}
	err := Transaction(ctx, func(txCtx context.Context) error {
		db := dbFrom(txCtx)
		var template model.VideoTemplate
		query := db.Select("id", "favorite_count").Where("id = ?", templateID)
		if favorited {
			query = query.Where("status = ?", 1)
		}
		if err := query.First(&template).Error; err != nil {
			return err
		}

		changed := false
		if favorited {
			favorite := model.VideoUserTemplateFavorite{UserID: userID, TemplateID: templateID}
			result := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "template_id"}},
				DoNothing: true,
			}).Create(&favorite)
			if result.Error != nil {
				return result.Error
			}
			changed = result.RowsAffected > 0
			if changed {
				if err := db.Model(&model.VideoTemplate{}).Where("id = ?", templateID).
					UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).Error; err != nil {
					return err
				}
			}
		} else {
			result := db.Where("user_id = ? AND template_id = ?", userID, templateID).
				Delete(&model.VideoUserTemplateFavorite{})
			if result.Error != nil {
				return result.Error
			}
			changed = result.RowsAffected > 0
			if changed {
				if err := db.Model(&model.VideoTemplate{}).Where("id = ?", templateID).
					UpdateColumn("favorite_count", gorm.Expr("CASE WHEN favorite_count > 0 THEN favorite_count - 1 ELSE 0 END")).Error; err != nil {
					return err
				}
			}
		}

		if err := db.Model(&model.VideoTemplate{}).Select("favorite_count").Where("id = ?", templateID).
			Scan(&state.FavoriteCount).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return state, nil
}

package repository

import (
	"context"
	"fmt"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type PointsPackageRepo struct {
	BaseRepo[model.VideoPointsPackage]
}

func NewPointsPackageRepo() *PointsPackageRepo { return &PointsPackageRepo{} }

type PointsPackageListFilter struct {
	PackageID    uint64
	ChannelID    uint64
	System       string
	UserType     int
	ResourceType string
	Status       *int8
	Keyword      string
}

func (r *PointsPackageRepo) PageList(ctx context.Context, page, pageSize int, filter *PointsPackageListFilter) ([]model.VideoPointsPackage, int64, error) {
	q := qFrom(ctx)
	points := q.VideoPointsPackage
	dao := points.WithContext(ctx)
	if filter != nil {
		if filter.PackageID != 0 {
			codes, err := r.productCodesByPackageID(ctx, filter.PackageID)
			if err != nil {
				return nil, 0, err
			}
			if len(codes) == 0 {
				return []model.VideoPointsPackage{}, 0, nil
			}
			dao = dao.Where(points.ProductCode.In(codes...))
		}
		if filter.ChannelID != 0 {
			codes, err := r.productCodesByChannelID(ctx, filter.ChannelID)
			if err != nil {
				return nil, 0, err
			}
			if len(codes) == 0 {
				return []model.VideoPointsPackage{}, 0, nil
			}
			dao = dao.Where(points.ProductCode.In(codes...))
		}
		if filter.System != "" {
			dao = dao.Where(points.Systems.Like("%\"" + filter.System + "\"%"))
		}
		if filter.UserType != 0 {
			dao = dao.Where(points.UserTypes.Like("%" + fmt.Sprint(filter.UserType) + "%"))
		}
		if filter.ResourceType != "" {
			dao = dao.Where(points.ResourceType.Eq(filter.ResourceType))
		}
		if filter.Status != nil {
			dao = dao.Where(points.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				points.ProductCode.Like(keyword), points.Name.Like(keyword),
				points.BadgeText.Like(keyword), points.Description.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Preload(points.Packages, points.Channels).
		Order(points.Sort.Asc(), points.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *PointsPackageRepo) productCodesByPackageID(ctx context.Context, packageID uint64) ([]string, error) {
	q := qFrom(ctx)
	appPackage, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode).
		Where(q.VideoPackage.ID.Eq(packageID)).First()
	if err != nil {
		return nil, err
	}
	relation := q.VideoPointsPackagePackage
	var codes []string
	err = relation.WithContext(ctx).Where(relation.PackageCode.Eq(appPackage.PackageCode)).
		Pluck(relation.ProductCode, &codes)
	return codes, err
}

func (r *PointsPackageRepo) productCodesByChannelID(ctx context.Context, channelID uint64) ([]string, error) {
	q := qFrom(ctx)
	channel, err := q.VideoChannel.WithContext(ctx).Select(q.VideoChannel.ChannelCode).
		Where(q.VideoChannel.ChannelID.Eq(channelID)).First()
	if err != nil {
		return nil, err
	}
	relation := q.VideoPointsPackageChannel
	var codes []string
	err = relation.WithContext(ctx).Where(relation.ChannelCode.Eq(channel.ChannelCode)).
		Pluck(relation.ProductCode, &codes)
	return codes, err
}

func (r *PointsPackageRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoPointsPackage, error) {
	q := qFrom(ctx).VideoPointsPackage
	return q.WithContext(ctx).Preload(q.Packages, q.Channels).Where(q.ID.Eq(id)).First()
}

func (r *PointsPackageRepo) GetByProductID(ctx context.Context, productID string) (*model.VideoPointsPackage, error) {
	q := qFrom(ctx).VideoPointsPackage
	return q.WithContext(ctx).Where(q.ProductCode.Eq(productID)).First()
}

// GetAppleProduct 按商店 SKU 和调用方包名解析启用的积分商品。
func (r *PointsPackageRepo) GetAppleProduct(ctx context.Context, productID, packageCode string) (*model.VideoPointsPackage, error) {
	q := qFrom(ctx)
	relation := q.VideoPointsPackagePackage
	if _, err := relation.WithContext(ctx).Where(
		relation.ProductCode.Eq(productID), relation.PackageCode.Eq(packageCode),
	).First(); err != nil {
		return nil, err
	}
	points := q.VideoPointsPackage
	return points.WithContext(ctx).Where(points.ProductCode.Eq(productID), points.Status.Eq(1)).First()
}

func (r *PointsPackageRepo) ListOptions(ctx context.Context) ([]model.VideoPointsPackage, error) {
	q := qFrom(ctx).VideoPointsPackage
	rows, err := q.WithContext(ctx).Preload(q.Packages, q.Channels).
		Where(q.Status.Eq(1)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *PointsPackageRepo) UpdateFields(ctx context.Context, item *model.VideoPointsPackage) error {
	q := qFrom(ctx).VideoPointsPackage
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.ProductCode, q.Name, q.Systems, q.UserTypes, q.ResourceType, q.Points,
		q.Currency, q.SalePrice, q.ActualRevenue, q.OriginalPrice, q.BadgeText,
		q.Description, q.ButtonText, q.IsDefault, q.Status, q.Sort,
	).Updates(item)
	return err
}

func (r *PointsPackageRepo) ReplaceTargets(ctx context.Context, item *model.VideoPointsPackage, packageID uint64, channelIDs []uint64) error {
	q := qFrom(ctx)
	appPackage, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode).
		Where(q.VideoPackage.ID.Eq(packageID)).First()
	if err != nil {
		return err
	}
	channels := []*model.VideoChannel{}
	if len(channelIDs) > 0 {
		channel := q.VideoChannel
		channels, err = channel.WithContext(ctx).Select(channel.ChannelID, channel.ChannelCode).
			Where(channel.ChannelID.In(channelIDs...)).Find()
		if err != nil {
			return err
		}
		if len(channels) != len(channelIDs) {
			return fmt.Errorf("one or more channels do not exist")
		}
	}
	packageRelation := q.VideoPointsPackagePackage
	if _, err := packageRelation.WithContext(ctx).Unscoped().Where(
		packageRelation.ProductCode.Eq(item.ProductCode),
	).Delete(); err != nil {
		return err
	}
	now := time.Now()
	if err := packageRelation.WithContext(ctx).Create(&model.VideoPointsPackagePackage{
		ProductCode: item.ProductCode, PackageCode: appPackage.PackageCode,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}
	channelRelation := q.VideoPointsPackageChannel
	if _, err := channelRelation.WithContext(ctx).Unscoped().Where(
		channelRelation.ProductCode.Eq(item.ProductCode),
	).Delete(); err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}
	rows := make([]*model.VideoPointsPackageChannel, 0, len(channels))
	for _, channel := range channels {
		rows = append(rows, &model.VideoPointsPackageChannel{
			ProductCode: item.ProductCode, ChannelCode: channel.ChannelCode,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	return channelRelation.WithContext(ctx).Create(rows...)
}

func (r *PointsPackageRepo) ClearDefaults(ctx context.Context, packageID uint64, resourceType string, exceptID uint64) error {
	codes, err := r.productCodesByPackageID(ctx, packageID)
	if err != nil || len(codes) == 0 {
		return err
	}
	q := qFrom(ctx).VideoPointsPackage
	dao := q.WithContext(ctx).Where(
		q.ProductCode.In(codes...), q.ResourceType.Eq(resourceType), q.IsDefault.Eq(1),
	)
	if exceptID != 0 {
		dao = dao.Where(q.ID.Neq(exceptID))
	}
	_, err = dao.Update(q.IsDefault, int8(0))
	return err
}

func (r *PointsPackageRepo) SetDefault(ctx context.Context, item *model.VideoPointsPackage) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		relation := q.VideoPointsPackagePackage
		target, err := relation.WithContext(txCtx).Where(relation.ProductCode.Eq(item.ProductCode)).First()
		if err != nil {
			return fmt.Errorf("points package is not associated with an app package: %w", err)
		}
		appPackage, err := q.VideoPackage.WithContext(txCtx).Select(q.VideoPackage.ID).
			Where(q.VideoPackage.PackageCode.Eq(target.PackageCode)).First()
		if err != nil {
			return err
		}
		if err := r.ClearDefaults(txCtx, appPackage.ID, item.ResourceType, item.ID); err != nil {
			return err
		}
		points := q.VideoPointsPackage
		_, err = points.WithContext(txCtx).Where(points.ID.Eq(item.ID)).Update(points.IsDefault, int8(1))
		return err
	})
}

func (r *PointsPackageRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	q := qFrom(ctx).VideoPointsPackage
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.Status, status)
	return err
}

func (r *PointsPackageRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		points := q.VideoPointsPackage
		item, err := points.WithContext(txCtx).Select(points.ID, points.ProductCode).Where(points.ID.Eq(id)).First()
		if err != nil {
			return err
		}
		packageRelation := q.VideoPointsPackagePackage
		if _, err := packageRelation.WithContext(txCtx).Unscoped().Where(
			packageRelation.ProductCode.Eq(item.ProductCode),
		).Delete(); err != nil {
			return err
		}
		channelRelation := q.VideoPointsPackageChannel
		if _, err := channelRelation.WithContext(txCtx).Unscoped().Where(
			channelRelation.ProductCode.Eq(item.ProductCode),
		).Delete(); err != nil {
			return err
		}
		_, err = points.WithContext(txCtx).Where(points.ID.Eq(id)).Delete()
		return err
	})
}

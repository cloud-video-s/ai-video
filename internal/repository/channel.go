package repository

import (
	"context"
	"strconv"
	"strings"

	"ai-video/internal/gen/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gen/field"
)

type ChannelRepo struct {
	BaseRepo[model.VideoChannel]
}

// ChannelRecord adds display names used by channel management to the generated
// channel model. Persistence remains owned by model.VideoChannel.
type ChannelRecord struct {
	model.VideoChannel
	AdMedia             string `json:"ad_media"`
	DeliveryPackageName string `json:"delivery_package_name"`
	OwnerUsername       string `json:"owner_username"`
	OwnerNickname       string `json:"owner_nickname"`
}

func (r *ChannelRepo) ResolveEnabledTargets(ctx *gin.Context, codeOrID, deliveryPackage string) ([]model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	dao := q.WithContext(ctx).Where(q.Status.Eq(1))
	if value := strings.TrimSpace(codeOrID); value != "" {
		if id, err := strconv.ParseUint(value, 10, 64); err == nil && id > 0 {
			dao = dao.Where(q.ID.Eq(id))
		} else {
			dao = dao.Where(q.ChannelCode.Eq(value))
		}
	}
	if value := strings.TrimSpace(deliveryPackage); value != "" {
		dao = dao.Where(q.DeliveryPackage.Eq(value))
	}
	if strings.TrimSpace(codeOrID) == "" && strings.TrimSpace(deliveryPackage) == "" {
		return []model.VideoChannel{}, nil
	}
	rows, err := dao.Order(q.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	return valuesOf(rows), nil
}

func NewChannelRepo() *ChannelRepo {
	return &ChannelRepo{}
}

type ChannelListFilter struct {
	ListSort        ListSort
	AgencyCompany   string
	DeliveryPackage string
	AdPlatform      string
	UploadMethod    string
	Status          *int8
	Keyword         string
}

func (r *ChannelRepo) PageList(ctx context.Context, page, pageSize int, filter *ChannelListFilter) ([]ChannelRecord, int64, error) {
	q := qFrom(ctx).VideoChannel
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.AgencyCompany != "" {
			dao = dao.Where(q.AgencyCompany.Like("%" + filter.AgencyCompany + "%"))
		}
		if filter.DeliveryPackage != "" {
			dao = dao.Where(q.DeliveryPackage.Eq(filter.DeliveryPackage))
		}
		if filter.AdPlatform != "" {
			dao = dao.Where(q.AdPlatform.Like("%" + filter.AdPlatform + "%"))
		}
		if filter.UploadMethod != "" {
			dao = dao.Where(q.UploadMethod.Eq(filter.UploadMethod))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				q.ChannelCode.Like(keyword),
				q.ChannelName.Like(keyword),
				field.NewUnsafeFieldRaw("CAST(video_channel.id AS CHAR) LIKE ?", keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	listSort := ListSort{}
	if filter != nil {
		listSort = filter.ListSort
	}
	order := orderForList(listSort, map[string]field.OrderExpr{"id": q.ID}, q.ID, q.ID.Desc())
	rows, err := dao.Order(order...).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	records, err := r.loadRecords(ctx, valuesOf(rows))
	return records, total, err
}

func (r *ChannelRepo) ListOptions(ctx context.Context) ([]model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.ChannelName.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *ChannelRepo) GetRecordByID(ctx context.Context, id uint64) (*ChannelRecord, error) {
	q := qFrom(ctx).VideoChannel
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoChannel{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *ChannelRepo) CreateRecord(ctx context.Context, item *ChannelRecord) error {
	q := qFrom(ctx).VideoChannel
	err := q.WithContext(ctx).Select(
		q.ChannelCode, q.ChannelName, q.AccountChannel, q.AgencyCompany, q.MediaID, q.AdPlatform,
		q.DeliveryPackage, q.SystemType, q.OwnerAdminID, q.AdAccount, q.TrackingURL, q.LandingPage,
		q.PortRebate, q.ServiceOrderFee, q.UploadMethod, q.CallbackConfig, q.Status, q.CreatedAt, q.UpdatedAt,
	).Create(&item.VideoChannel)
	return err
}

func (r *ChannelRepo) UpdateRecord(ctx context.Context, item *ChannelRecord) error {
	q := qFrom(ctx).VideoChannel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.ChannelName, q.AccountChannel, q.AgencyCompany, q.MediaID, q.AdPlatform,
		q.DeliveryPackage, q.SystemType, q.OwnerAdminID, q.AdAccount, q.TrackingURL,
		q.LandingPage, q.PortRebate, q.ServiceOrderFee, q.UploadMethod, q.CallbackConfig, q.Status, q.UpdatedAt,
	).Updates(&item.VideoChannel)
	return err
}

func (r *ChannelRepo) loadRecords(ctx context.Context, items []model.VideoChannel) ([]ChannelRecord, error) {
	records := make([]ChannelRecord, len(items))
	if len(items) == 0 {
		return records, nil
	}
	packageCodes := make([]string, 0, len(items))
	mediaIDs := make([]uint64, 0, len(items))
	ownerIDs := make([]uint64, 0, len(items))
	seenPackages := make(map[string]struct{}, len(items))
	seenMedia := make(map[uint64]struct{}, len(items))
	seenOwners := make(map[uint64]struct{}, len(items))
	for i := range items {
		records[i].VideoChannel = items[i]
		if code := items[i].DeliveryPackage; code != "" {
			if _, exists := seenPackages[code]; !exists {
				seenPackages[code] = struct{}{}
				packageCodes = append(packageCodes, code)
			}
		}
		if id := items[i].MediaID; id != 0 {
			if _, exists := seenMedia[id]; !exists {
				seenMedia[id] = struct{}{}
				mediaIDs = append(mediaIDs, id)
			}
		}
		if items[i].OwnerAdminID != nil && *items[i].OwnerAdminID != 0 {
			id := *items[i].OwnerAdminID
			if _, exists := seenOwners[id]; !exists {
				seenOwners[id] = struct{}{}
				ownerIDs = append(ownerIDs, id)
			}
		}
	}

	q := qFrom(ctx)
	packageNames := make(map[string]string, len(packageCodes))
	if len(packageCodes) > 0 {
		rows, err := q.VideoPackage.WithContext(ctx).Where(q.VideoPackage.PackageCode.In(packageCodes...)).Find()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			packageNames[row.PackageCode] = row.PackageName
		}
	}
	mediaNames := make(map[uint64]string, len(mediaIDs))
	if len(mediaIDs) > 0 {
		rows, err := q.VideoMedium.WithContext(ctx).Where(q.VideoMedium.ID.In(mediaIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			mediaNames[row.ID] = row.Name
		}
	}
	type ownerName struct {
		Username string
		Nickname string
	}
	ownerNames := make(map[uint64]ownerName, len(ownerIDs))
	if len(ownerIDs) > 0 {
		rows, err := q.VideoAdmin.WithContext(ctx).Where(q.VideoAdmin.ID.In(ownerIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			ownerNames[row.ID] = ownerName{Username: row.Username, Nickname: row.Nickname}
		}
	}
	for i := range records {
		records[i].DeliveryPackageName = packageNames[records[i].DeliveryPackage]
		records[i].AdMedia = mediaNames[records[i].MediaID]
		if records[i].OwnerAdminID != nil {
			owner := ownerNames[*records[i].OwnerAdminID]
			records[i].OwnerUsername = owner.Username
			records[i].OwnerNickname = owner.Nickname
		}
	}
	return records, nil
}

func (r *ChannelRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	q := qFrom(ctx).VideoChannel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.Status, status)
	return err
}

func (r *ChannelRepo) GetByCode(ctx context.Context, code string) (*model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	return q.WithContext(ctx).Where(q.ChannelCode.Eq(code)).First()
}

func (r *ChannelRepo) GetByCodeOrID(ctx context.Context, value string) (*model.VideoChannel, error) {
	if id, err := strconv.ParseUint(value, 10, 64); err == nil && id > 0 {
		q := qFrom(ctx).VideoChannel
		if item, getErr := q.WithContext(ctx).Where(q.ID.Eq(id)).First(); getErr == nil {
			return item, nil
		}
	}
	return r.GetByCode(ctx, value)
}

func (r *ChannelRepo) TemplateCount(_ context.Context, _ uint64) (int64, error) {
	// The current template model no longer has a direct channel relationship.
	return 0, nil
}

func (r *ChannelRepo) GetByAdAccountID(ctx context.Context, adAccount, mediaID uint64) (uint64, error) {
	q := qFrom(ctx).VideoChannel
	account := strconv.FormatUint(adAccount, 10)
	accountVariants := []string{account, "act_" + account, "ACT_" + account}
	if len(account) == 10 {
		accountVariants = append(accountVariants, account[:3]+"-"+account[3:6]+"-"+account[6:])
	}
	dao := q.WithContext(ctx).Where(q.Status.Eq(1), q.AdAccount.In(accountVariants...))
	if mediaID != 0 {
		dao = dao.Where(q.MediaID.Eq(mediaID))
	}
	find, err := dao.Order(q.ID.Asc()).First()
	if err != nil {
		return 0, err
	}
	return find.ID, nil
}

func (r *ChannelRepo) GetEnabledByID(ctx context.Context, id uint64) (*model.VideoChannel, error) {
	q := qFrom(ctx).VideoChannel
	return q.WithContext(ctx).Where(q.ID.Eq(id), q.Status.Eq(1)).First()
}

func (r *ChannelRepo) GetDefaultID(ctx context.Context) (uint64, error) {
	q := qFrom(ctx).VideoChannel
	channel, err := q.WithContext(ctx).
		Where(q.Status.Eq(1), q.IsDefault.Eq(1)).
		Order(q.ID.Asc()).
		First()
	if err != nil {
		return 0, err
	}
	return channel.ID, nil
}

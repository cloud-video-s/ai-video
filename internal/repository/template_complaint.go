package repository

import (
	"context"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// TemplateComplaintAdminFilter contains the read-only filters exposed by the
// operations complaint-management page.
type TemplateComplaintAdminFilter struct {
	UserID        uint64
	TemplateID    uint64
	ComplaintType string
	Keyword       string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
}

// TemplateComplaintRepo manages user complaints about video templates.
type TemplateComplaintRepo struct{}

func NewTemplateComplaintRepo() *TemplateComplaintRepo {
	return &TemplateComplaintRepo{}
}

// PageAdmin lists active complaints and preloads their user and template.
// Related rows are loaded unscoped so historical complaints remain useful
// after a user or template has been soft-deleted.
func (r *TemplateComplaintRepo) PageAdmin(
	ctx context.Context,
	page, pageSize int,
	filter *TemplateComplaintAdminFilter,
) ([]model.VideoUserTemplateComplaint, int64, error) {
	complaintTable := model.TableNameVideoUserTemplateComplaint
	userTable := model.TableNameVideoUser
	templateTable := model.TableNameVideoTemplate

	dao := qFrom(ctx).UnderlyingDB().Model(&model.VideoUserTemplateComplaint{}).
		Joins("LEFT JOIN " + userTable + " ON " + userTable + ".id = " + complaintTable + ".user_id").
		Joins("LEFT JOIN " + templateTable + " ON " + templateTable + ".id = " + complaintTable + ".template_id")
	if filter != nil {
		if filter.UserID != 0 {
			dao = dao.Where(complaintTable+".user_id = ?", filter.UserID)
		}
		if filter.TemplateID != 0 {
			dao = dao.Where(complaintTable+".template_id = ?", filter.TemplateID)
		}
		if filter.ComplaintType != "" {
			dao = dao.Where(complaintTable+".complaint_type = ?", filter.ComplaintType)
		}
		if filter.CreatedFrom != nil {
			dao = dao.Where(complaintTable+".created_at >= ?", *filter.CreatedFrom)
		}
		if filter.CreatedTo != nil {
			dao = dao.Where(complaintTable+".created_at < ?", *filter.CreatedTo)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where("("+
				complaintTable+".complaint_type LIKE ? OR "+complaintTable+".content LIKE ? OR "+
				templateTable+".name LIKE ? OR "+
				userTable+".username LIKE ? OR "+userTable+".login_account LIKE ? OR "+
				userTable+".email LIKE ? OR "+userTable+".phone LIKE ? OR "+
				userTable+".imei LIKE ? OR "+userTable+".device_code LIKE ?)",
				keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword,
			)
		}
	}

	var total int64
	if err := dao.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var complaints []model.VideoUserTemplateComplaint
	if err := preloadTemplateComplaintAdminRelations(dao).
		Select(complaintTable + ".*").
		Order(complaintTable + ".created_at DESC").
		Order(complaintTable + ".id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&complaints).Error; err != nil {
		return nil, 0, err
	}
	return complaints, total, nil
}

// GetAdminDetail returns one active complaint with its historical relations.
func (r *TemplateComplaintRepo) GetAdminDetail(ctx context.Context, id uint64) (*model.VideoUserTemplateComplaint, error) {
	var complaint model.VideoUserTemplateComplaint
	dao := qFrom(ctx).UnderlyingDB().Model(&model.VideoUserTemplateComplaint{})
	if err := preloadTemplateComplaintAdminRelations(dao).
		Where(model.TableNameVideoUserTemplateComplaint+".id = ?", id).
		First(&complaint).Error; err != nil {
		return nil, err
	}
	return &complaint, nil
}

func preloadTemplateComplaintAdminRelations(dao *gorm.DB) *gorm.DB {
	preloadUnscoped := func(relation *gorm.DB) *gorm.DB { return relation.Unscoped() }
	return dao.Preload("User", preloadUnscoped).Preload("Template", preloadUnscoped)
}

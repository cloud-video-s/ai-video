package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
)

type TemplateComplaintService struct {
	repo *repository.TemplateComplaintRepo
}

func NewTemplateComplaintService() *TemplateComplaintService {
	return &TemplateComplaintService{repo: repository.NewTemplateComplaintRepo()}
}

type ListTemplateComplaintRequest struct {
	UserID        uint64 `form:"user_id"`
	TemplateID    uint64 `form:"template_id"`
	ComplaintType string `form:"complaint_type" binding:"max=255"`
	Keyword       string `form:"keyword" binding:"max=255"`
	DateFrom      string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo        string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
}

type TemplateComplaintUserView struct {
	ID           uint64 `json:"id"`
	Username     string `json:"username"`
	LoginAccount string `json:"login_account"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	IMEI         string `json:"imei"`
	DeviceCode   string `json:"device_code"`
	Status       int8   `json:"status"`
	Deleted      bool   `json:"deleted"`
}

type TemplateComplaintTemplateView struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	TemplateType  int64  `json:"template_type"`
	CoverImageURL string `json:"cover_image_url"`
	OriginalURL   string `json:"original_url"`
	ThumbnailURL  string `json:"thumbnail_url"`
	Status        int32  `json:"status"`
	Deleted       bool   `json:"deleted"`
}

type TemplateComplaintView struct {
	ID            uint64                         `json:"id"`
	UserID        uint64                         `json:"user_id"`
	TemplateID    uint64                         `json:"template_id"`
	ComplaintType string                         `json:"complaint_type"`
	Content       string                         `json:"content"`
	CreatedAt     time.Time                      `json:"created_at"`
	UpdatedAt     time.Time                      `json:"updated_at"`
	User          *TemplateComplaintUserView     `json:"user"`
	Template      *TemplateComplaintTemplateView `json:"template"`
}

func (s *TemplateComplaintService) List(
	ctx context.Context,
	page, pageSize int,
	req *ListTemplateComplaintRequest,
) ([]TemplateComplaintView, int64, error) {
	from, to, err := parseTemplateComplaintDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, 0, err
	}
	records, total, err := s.repo.PageAdmin(ctx, page, pageSize, &repository.TemplateComplaintAdminFilter{
		UserID: req.UserID, TemplateID: req.TemplateID,
		ComplaintType: strings.TrimSpace(req.ComplaintType),
		Keyword:       strings.TrimSpace(req.Keyword), CreatedFrom: from, CreatedTo: to,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]TemplateComplaintView, 0, len(records))
	for i := range records {
		items = append(items, templateComplaintView(&records[i]))
	}
	return items, total, nil
}

func (s *TemplateComplaintService) GetByID(ctx context.Context, id uint64) (*TemplateComplaintView, error) {
	record, err := s.repo.GetAdminDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "投诉记录不存在")
	}
	view := templateComplaintView(record)
	return &view, nil
}

func templateComplaintView(complaint *model.VideoUserTemplateComplaint) TemplateComplaintView {
	view := TemplateComplaintView{
		ID: complaint.ID, UserID: complaint.UserID, TemplateID: complaint.TemplateID,
		ComplaintType: complaint.ComplaintType, Content: complaint.Content,
		CreatedAt: complaint.CreatedAt, UpdatedAt: complaint.UpdatedAt,
	}
	if complaint.User.ID != 0 {
		view.User = &TemplateComplaintUserView{
			ID: complaint.User.ID, Username: complaint.User.Username,
			LoginAccount: complaint.User.LoginAccount, Email: complaint.User.Email,
			Phone: complaint.User.Phone, IMEI: complaint.User.IMEI,
			DeviceCode: complaint.User.DeviceCode, Status: complaint.User.Status,
			Deleted: complaint.User.DeletedAt.Valid,
		}
	}
	if complaint.Template.ID != 0 {
		view.Template = &TemplateComplaintTemplateView{
			ID: complaint.Template.ID, Name: complaint.Template.Name,
			TemplateType:  complaint.Template.TemplateType,
			CoverImageURL: complaint.Template.CoverImageURL,
			OriginalURL:   complaint.Template.OriginalURL,
			ThumbnailURL:  complaint.Template.ThumbnailURL,
			Status:        complaint.Template.Status, Deleted: complaint.Template.DeletedAt.Valid,
		}
	}
	return view
}

func parseTemplateComplaintDateRange(fromValue, toValue string) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if fromValue != "" {
		value, err := time.ParseInLocation("2006-01-02", fromValue, time.Local)
		if err != nil {
			return nil, nil, errors.New("开始日期格式错误")
		}
		from = &value
	}
	if toValue != "" {
		value, err := time.ParseInLocation("2006-01-02", toValue, time.Local)
		if err != nil {
			return nil, nil, errors.New("结束日期格式错误")
		}
		value = value.AddDate(0, 0, 1)
		to = &value
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, errors.New("开始日期不能晚于结束日期")
	}
	return from, to, nil
}

package service

import (
	"ai-video/internal/middleware"
	"fmt"
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/uploadruntime"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type ClientBannerService struct {
	bannerRepo *repository.BannerRepo
	userRepo   *repository.AppUserRepo
}

func NewClientBannerService() *ClientBannerService {
	return &ClientBannerService{
		bannerRepo: repository.NewBannerRepo(), userRepo: repository.NewAppUserRepo(),
	}
}

type ClientBannerRequest struct {
	PositionKey string `form:"position_key" binding:"required,max=100"`
	AccountBaseRequest
}

type ClientBannerTemplate struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	TemplateType  int64  `json:"template_type"`
	CoverImageURL string `json:"cover_image_url"`
	OriginalURL   string `json:"original_url"`
	ThumbnailURL  string `json:"thumbnail_url"`
	Status        int32  `json:"status"`
}

type ClientBanner struct {
	ID             uint64                `json:"id"`
	Name           string                `json:"name"`
	PositionKey    string                `json:"position_key"`
	Status         int8                  `json:"status"`
	JumpType       uint8                 `json:"jump_type"`
	CoverImage     string                `json:"cover_image"`
	Route          string                `json:"route"`
	TargetTemplate *ClientBannerTemplate `json:"target_template,omitempty"`
	TemplateID     *uint64               `json:"template_id,omitempty"`
	Sort           uint64                `json:"sort"`
}

func (s *ClientBannerService) List(ctx *gin.Context, req *ClientBannerRequest) ([]ClientBanner, error) {
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	countryCode := req.ClientCountry
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.ClientCountry))
	}

	membershipStatus := uint8(1)
	if user.SubscriptionStatus == 2 {
		membershipStatus = 2
	}
	rows, err := s.bannerRepo.ListForClient(ctx, repository.ClientBannerTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
		AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
		VersionCode:        strings.TrimSpace(req.AppVersion),
		SubscriptionStatus: membershipStatus,
	})
	if err != nil {
		return nil, err
	}
	result := make([]ClientBanner, 0, len(rows))
	for i := range rows {
		item := mapClientBanner(&rows[i])
		item.PositionKey = strings.TrimSpace(req.PositionKey)
		result = append(result, item)
	}
	return result, nil
}

func mapClientBanner(item *model.VideoBanner) ClientBanner {
	positionKey := ""
	result := ClientBanner{
		ID: item.ID, Name: item.Name, PositionKey: positionKey,
		Status: item.Status, JumpType: item.JumpType,
		CoverImage: uploadruntime.PublicURL(item.CoverImage), Route: clientBannerRoute(item), TemplateID: item.TemplateID, Sort: item.Sort,
	}
	if item.Template.ID != 0 {
		result.TargetTemplate = &ClientBannerTemplate{
			ID: item.Template.ID, Name: item.Template.Name, TemplateType: item.Template.TemplateType,
			CoverImageURL: uploadruntime.PublicURL(item.Template.CoverImageURL),
			OriginalURL:   uploadruntime.PublicURL(item.Template.OriginalURL),
			ThumbnailURL:  uploadruntime.PublicURL(item.Template.ThumbnailURL), Status: item.Template.Status,
		}
	}
	return result
}

func clientBannerPositionKeys(items []model.VideoDisplayPosition) []string {
	result := make([]string, 0, len(items))
	for i := range items {
		if key := strings.TrimSpace(items[i].PositionKey); key != "" {
			result = append(result, key)
		}
	}
	return result
}

func clientBannerRoute(item *model.VideoBanner) string {
	if item.JumpURL != "" {
		return item.JumpURL
	}
	switch item.JumpType {
	case domain.BannerJumpTypeTemplate:
		if item.TemplateID != nil {
			return fmt.Sprintf("/templates/%d", *item.TemplateID)
		}
	case domain.BannerJumpTypeTextToImage:
		return "/text-to-image"
	case domain.BannerJumpTypeTextToVideo:
		return "/text-to-video"
	}
	return ""
}

func clientChannelIDs(items []model.VideoChannel) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ChannelID
	}
	return result
}

func clientPackageIDs(items []model.VideoPackage) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func firstNotEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

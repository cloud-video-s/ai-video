package service

import (
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/middleware"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type ClientPointProductService struct {
	pointsRepo *repository.PointsRepo
	userRepo   *repository.AppUserRepo
}

func NewClientPointProductService() *ClientPointProductService {
	return &ClientPointProductService{
		pointsRepo: repository.NewPointsRepo(),
		userRepo:   repository.NewAppUserRepo(),
	}
}

type ClientPointProductResponse struct {
	ID            uint64  `json:"id"`
	ProductCode   string  `json:"product_code"`
	Name          string  `json:"name"`
	ResourceType  string  `json:"resource_type"`
	Points        uint64  `json:"points"`
	Currency      string  `json:"currency"`
	SalePrice     float64 `json:"sale_price"`
	OriginalPrice float64 `json:"original_price"`
	Icon          string  `json:"icon"`
	Description   string  `json:"description"`
	ButtonText    string  `json:"button_text"`
	IsDefault     bool    `json:"is_default"`
	Status        int8    `json:"status"`
	Sort          int64   `json:"sort"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

func (s *ClientPointProductService) List(ctx *gin.Context) ([]ClientPointProductResponse, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}

	countryCode := strings.ToUpper(strings.TrimSpace(middleware.GetAPIDeviceCountry(ctx)))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.ClientCountry))
	}
	items, err := s.pointsRepo.ListForClient(ctx, repository.ClientPointsTargets{
		AppCode:     strings.TrimSpace(middleware.GetAPIAPPCode(ctx)),
		PackageCode: strings.TrimSpace(middleware.GetAPIAppPackageCode(ctx)),
		VersionCode: strings.TrimSpace(middleware.GetAPIAppVersion(ctx)),
		CountryCode: countryCode,
		ChannelCode: strings.TrimSpace(middleware.GetAPIChannelCode(ctx)),
		System:      clientPointProductSystem(middleware.GetAPISystemType(ctx)),
		UserType:    int(user.UserType),
	})
	if err != nil {
		return nil, err
	}

	result := make([]ClientPointProductResponse, 0, len(items))
	for i := range items {
		result = append(result, mapClientPointProduct(items[i]))
	}
	return result, nil
}

func clientPointProductSystem(systemType int) string {
	switch systemType {
	case domain.SystemTypeIos:
		return "ios"
	case domain.SystemTypeA:
		return "android"
	default:
		return ""
	}
}

func mapClientPointProduct(item *model.VideoPoint) ClientPointProductResponse {
	return ClientPointProductResponse{
		ID: item.ID, ProductCode: item.ProductCode, Name: item.Name,
		ResourceType: item.ResourceType,
		Points:       item.Points, Currency: item.Currency, SalePrice: item.SalePrice,
		OriginalPrice: item.OriginalPrice, Icon: item.Icon,
		Description: item.Description, ButtonText: item.ButtonText,
		IsDefault: item.IsDefault == 1, Status: item.Status, Sort: item.Sort,
		CreatedAt: item.CreatedAt.Unix(), UpdatedAt: item.UpdatedAt.Unix(),
	}
}

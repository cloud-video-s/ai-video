package service

import (
	"ai-video/internal/middleware"
	"ai-video/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ClientVipService struct {
	subscriptionRepo *repository.VIPSubscriptionRepo
	userRepo         *repository.AppUserRepo
}

func NewClientVipService() *ClientVipService {
	return &ClientVipService{}
}

func (s *ClientVipService) Recommend(ctx *gin.Context, req *VipRecommendRequest) (interface{}, error) {
	//user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	//if err != nil {
	//	return nil, err
	//}

	return s.subscriptionRepo.Recommend(ctx, &repository.VIPSubscriptionListFilter{
		VipType:     strconv.Itoa(req.VipType),
		AppCode:     middleware.GetAPIAPPCode(ctx),
		PackageCode: middleware.GetAPIAppPackageCode(ctx),
		VersionCode: middleware.GetAPIAppVersion(ctx),
	})
}

type VipRecommendRequest struct {
	VipType int `json:"vip_type"`
	AccountBaseRequest
}

package service

import (
	"ai-video/internal/repository"
	"context"
)

type ClientMediaAdsService struct {
	pointsRepo *repository.PointsRepo
	userRepo   *repository.AppUserRepo
}

func NewClientMediaAdsService() *ClientMediaAdsService {
	return &ClientMediaAdsService{
		pointsRepo: repository.NewPointsRepo(),
		userRepo:   repository.NewAppUserRepo(),
	}
}

type MediaAdsService struct {
	TrackerToken string `json:"trackerToken"`
	TrackerName  string `json:"trackerName"`
	Network      string `json:"network"`
	Campaign     string `json:"campaign"`
	Creative     string `json:"creative"`
	Adgroup      string `json:"adgroup"`
}

func (r *ClientMediaAdsService) MediaAdsSave(ctx context.Context, info MediaAdsService) error {
	if info.Network == "" {

	}
	//if info.Campaign == "" {
	//	CampaignID := utils.ExtractBracketContent(info.Campaign)
	//}
	//if info.Creative == "" {
	//	CreativeID := utils.ExtractBracketContent(info.Creative)
	//}
	//if info.Adgroup == "" {
	//	AdgroupID := utils.ExtractBracketContent(info.Adgroup)
	//}
	return nil
}

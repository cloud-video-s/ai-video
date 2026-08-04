package service

import (
	"ai-video/internal/middleware"
	"ai-video/internal/repository"
	"time"

	"github.com/gin-gonic/gin"
)

type ClientPointsService struct {
	pointsRepo *repository.UserPointsLedgerRepo
	userRepo   *repository.AppUserRepo
}

func NewClientPointsService() *ClientPointsService {
	return &ClientPointsService{
		pointsRepo: repository.NewUserPointsLedgerRepo(), userRepo: repository.NewAppUserRepo(),
	}
}

type ClientPointsResponse struct {
	ID            uint64 `json:"id"`             // 主键ID
	UserID        uint64 `json:"user_id"`        // 客户端用户ID
	Direction     int8   `json:"direction"`      // 变动方向：1-收入，2-支出
	PointsChange  int64  `json:"points_change"`  // 积分变动量（正数表示增加，负数表示减少）
	BalanceBefore uint64 `json:"balance_before"` // 变动前积分余额
	BalanceAfter  uint64 `json:"balance_after"`  // 变动后积分余额
	Description   string `json:"description"`    // 变动描述（如“购买VIP月卡赠送”）
	CreatedAt     int64  `json:"created_at"`     // 记录创建时间（系统时间戳）
	UpdatedAt     int64  `json:"updated_at"`
}

type ClientPointsRequest struct {
	BasePage
	PointsType int8  `json:"points_type" query:"points_type" form:"points_type" binding:"omitempty,min=1" default:"1"`
	StartTime  int64 `json:"start_time" query:"start_time" form:"start_time" binding:"omitempty,min=1" default:"0"`
	EndTime    int64 `json:"end_time" query:"end_time" form:"end_time" binding:"omitempty,min=1" default:"0"`
}

func (h *ClientPointsService) GetPointsList(c *gin.Context, req ClientPointsRequest) (interface{}, error) {
	request := repository.UserPointsLedgerFilter{
		UserID:    middleware.GetAPIUserID(c),
		Direction: req.PointsType,
	}
	if req.StartTime != 0 && req.EndTime != 0 {
		startTime := time.Unix(req.StartTime, 0)
		endTime := time.Unix(req.EndTime, 0)
		request.OccurredFrom = &startTime
		request.OccurredTo = &endTime
	}
	list, total, err := h.pointsRepo.GetPointsList(c, req.Page, req.PageSize, &request)
	if err != nil {
		return nil, err
	}
	var data []ClientPointsResponse
	for _, item := range list {
		data = append(data, ClientPointsResponse{
			ID:            item.ID,
			UserID:        item.UserID,
			Direction:     item.Direction,
			PointsChange:  item.PointsChange,
			BalanceBefore: item.BalanceBefore,
			BalanceAfter:  item.BalanceAfter,
			Description:   item.Description,
			CreatedAt:     item.CreatedAt.Unix(),
			UpdatedAt:     item.UpdatedAt.Unix(),
		})
	}
	return GetPageResponse(int64(req.Page), int64(req.PageSize), total, data)
}

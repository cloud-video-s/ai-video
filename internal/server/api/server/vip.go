package service

import "github.com/gin-gonic/gin"

type ClientVipService struct {
}

func NewClientVipService() *ClientVipService {
	return &ClientVipService{}
}

func (s *ClientVipService) Recommend(ctx *gin.Context) (map[string]interface{}, error) {

	return nil, nil
}

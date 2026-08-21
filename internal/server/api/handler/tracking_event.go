package handler

import (
	"ai-video/internal/gen/model"
	"context"
	"errors"
	"net/http"

	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

const maxTrackingEventBodyBytes = 8 << 10

type trackingEventService interface {
	Report(
		ctx context.Context,
		userID uint64,
		client apiservice.TrackingEventClientContext,
		req apiservice.TrackingEventRequest,
	) (*model.VideoTrackingEvent, error)
}

type TrackingEventHandler struct {
	service trackingEventService
}

func NewTrackingEventHandler() *TrackingEventHandler {
	return &TrackingEventHandler{service: apiservice.NewTrackingEventService()}
}

// Report accepts one tracking event and appends one database record.
func (h *TrackingEventHandler) Report(c *gin.Context) {
	var req apiservice.TrackingEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, "invalid tracking event payload: "+err.Error())
		return
	}

	result, err := h.service.Report(
		c.Request.Context(),
		middleware.GetAPIUserID(c),
		apiservice.TrackingEventClientContext{
			AppCode:     middleware.GetAPIAPPCode(c),
			PackageCode: middleware.GetAPIAppPackageCode(c),
			AppVersion:  middleware.GetAPIAppVersion(c),
			ChannelCode: middleware.GetAPIChannelCode(c),
			CountryCode: middleware.GetAPIDeviceCountry(c),
			PhoneModel:  middleware.GetAPIPhoneModel(c),
			SystemType:  middleware.GetAPISystemType(c),
		},
		req,
	)
	if errors.Is(err, apiservice.ErrTrackingEventInvalid) {
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
		return
	}
	if err != nil {
		response.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, result)
}

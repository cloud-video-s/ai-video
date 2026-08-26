package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"ai-video/internal/generation"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GenerationHandler struct {
	manager      *generation.Manager
	authService  *apiservice.AuthService
	modelService *apiservice.GenerationModelService
}

func NewGenerationHandler() *GenerationHandler {
	return &GenerationHandler{
		manager:      generation.Shared(),
		modelService: apiservice.NewGenerationModelService(),
	}
}

// Models 按模型类型返回平台和模型均启用的模型及其参数。
func (h *GenerationHandler) Models(c *gin.Context) {
	var request apiservice.GenerationModelRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	items, err := h.modelService.List(c.Request.Context(), request.ModelType)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, items)
}

// Create 校验并创建一个属于当前客户端用户的待处理生成任务。
func (h *GenerationHandler) Create(c *gin.Context) {
	var request generation.CreateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	task, err := h.manager.CreateTask(c.Request.Context(), middleware.GetAPIUserID(c), &request)
	if err != nil {
		response.FailWithStatus(c, http.StatusBadGateway, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, generation.ViewOf(task))
}

// CreateFromTemplate derives the task type, model, prompt, and configured
// parameter defaults from an enabled template before queueing the task.
func (h *GenerationHandler) CreateFromTemplate(c *gin.Context) {
	var request generation.CreateTemplateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "invalid parameters: "+err.Error())
		return
	}
	task, err := h.manager.CreateTemplateTask(c.Request.Context(), middleware.GetAPIUserID(c), &request)
	if err != nil {
		if errors.Is(err, generation.ErrTemplateUnavailable) {
			response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
			return
		}
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, generation.ViewOf(task))
}

// CreateFromTool creates a task through a dedicated tool contract. The client
// supplies at most one image and one video; the tool owns the model, prompt,
// task type, and model parameter defaults.
func (h *GenerationHandler) CreateFromTool(c *gin.Context) {
	var request generation.CreateToolTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "invalid parameters: "+err.Error())
		return
	}
	task, err := h.manager.CreateToolTask(c, middleware.GetAPIUserID(c), &request)
	if err != nil {
		//if errors.Is(err, generation.ErrToolUnavailable) {
		//	response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
		//	return
		//}
		response.FailWithStatus(c, http.StatusBadRequest, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, generation.ViewOf(task))
}

// List 分页返回当前客户端用户自己的生成任务。
func (h *GenerationHandler) List(c *gin.Context) {
	var request apiservice.GenerationListRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		response.Fail(c, errcode.ErrParam, "参数错误: "+err.Error())
		return
	}
	if request.Page == 0 {
		request.Page = 1
	}
	if request.PageSize == 0 {
		request.PageSize = 10
	}
	data, err := h.modelService.GenerationList(c, &request)
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *GenerationHandler) Get(c *gin.Context) {
	taskID, ok := generationTaskID(c)
	if !ok {
		return
	}
	task, err := h.manager.GetTask(c.Request.Context(), middleware.GetAPIUserID(c), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
			return
		}
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	response.OK(c, generation.ViewOf(task))
}

func (h *GenerationHandler) Delete(c *gin.Context) {
	taskID, ok := generationTaskID(c)
	if !ok {
		return
	}
	if err := h.manager.DeleteTask(c.Request.Context(), middleware.GetAPIUserID(c), taskID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
			return
		}
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, nil)
}

// Events 建立 SSE 长连接并实时推送任务状态，任务终止后服务端主动结束连接。
func (h *GenerationHandler) Events(c *gin.Context) {
	task, err := h.manager.GetOngoingTask(c.Request.Context(), middleware.GetAPIUserID(c))
	if err != nil {
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, err.Error())
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	//var data []generation.TaskView
	//for _, item := range task {
	//	data = append(data, generation.ViewOf(item))
	//	h.manager.Publish()
	//
	//}
	c.SSEvent("task", task)
	c.Writer.Flush()
	events, unsubscribe := h.manager.Subscribe(task[0].ID)
	defer unsubscribe()
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event := <-events:
			c.SSEvent("task", event)
			c.Writer.Flush()
			if generation.IsTerminal(event.Status) {
				return
			}
		case at := <-heartbeat.C:
			c.SSEvent("heartbeat", gin.H{"time": at.Unix()})
			c.Writer.Flush()
		}
	}
}

func generationTaskID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, errcode.ErrParam, "任务 ID 无效")
		return 0, false
	}
	return id, true
}

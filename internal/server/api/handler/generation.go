package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"ai-video/internal/generation"
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/response"
	"ai-video/internal/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GenerationHandler struct{ manager *generation.Manager }

func NewGenerationHandler() *GenerationHandler {
	return &GenerationHandler{manager: generation.Shared()}
}

// Models 返回客户端当前可以使用的生成模型和默认参数。
func (h *GenerationHandler) Models(c *gin.Context) {
	items, err := h.manager.ListModels(c.Request.Context())
	if err != nil {
		response.Fail(c, errcode.ErrServer, err.Error())
		return
	}
	result := make([]gin.H, 0, len(items))
	for i := range items {
		var defaults map[string]interface{}
		_ = json.Unmarshal([]byte(items[i].DefaultParameters), &defaults)
		result = append(result, gin.H{
			"code": items[i].Code, "name": items[i].Name, "provider": items[i].Provider,
			"model_name": items[i].ModelName, "description": items[i].Description,
			"default_parameters": defaults,
		})
	}
	response.OK(c, result)
}

// Create 创建并立即提交一个属于当前客户端用户的视频生成任务。
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

// List 分页返回当前客户端用户自己的生成任务。
func (h *GenerationHandler) List(c *gin.Context) {
	pagination := utils.GetPagination(c)
	items, total, err := h.manager.ListTasks(
		c.Request.Context(), middleware.GetAPIUserID(c), pagination.Page, pagination.PageSize, c.Query("status"),
	)
	if err != nil {
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	list := make([]generation.TaskView, 0, len(items))
	for i := range items {
		list = append(list, generation.ViewOf(&items[i]))
	}
	response.OK(c, gin.H{"list": list, "total": total, "page": pagination.Page, "size": pagination.PageSize})
}

func (h *GenerationHandler) Get(c *gin.Context) {
	taskID, ok := generationTaskID(c)
	if !ok {
		return
	}
	task, err := h.manager.GetTask(c.Request.Context(), middleware.GetAPIUserID(c), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, "生成任务不存在")
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
			response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, "生成任务不存在")
			return
		}
		response.Fail(c, errcode.ErrParam, err.Error())
		return
	}
	response.OK(c, nil)
}

// Events 建立 SSE 长连接并实时推送任务状态，任务终止后服务端主动结束连接。
func (h *GenerationHandler) Events(c *gin.Context) {
	taskID, ok := generationTaskID(c)
	if !ok {
		return
	}
	task, err := h.manager.GetTask(c.Request.Context(), middleware.GetAPIUserID(c), taskID)
	if err != nil {
		response.FailWithStatus(c, http.StatusNotFound, errcode.ErrNotFound, "生成任务不存在")
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.SSEvent("task", generation.ViewOf(task))
	c.Writer.Flush()
	if generation.IsTerminal(task.Status) {
		return
	}
	events, unsubscribe := h.manager.Subscribe(taskID)
	defer unsubscribe()
	heartbeat := time.NewTicker(15 * time.Second)
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

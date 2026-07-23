package generation

import (
	"sync"

	"ai-video/internal/gen/model"
)

// Hub 管理当前进程中的 SSE 订阅，数据库仍是任务状态的最终事实来源。
type Hub struct {
	mu          sync.RWMutex
	subscribers map[uint64]map[chan TaskView]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[uint64]map[chan TaskView]struct{})}
}

func (h *Hub) Subscribe(taskID uint64) (<-chan TaskView, func()) {
	ch := make(chan TaskView, 8)
	h.mu.Lock()
	if h.subscribers[taskID] == nil {
		h.subscribers[taskID] = make(map[chan TaskView]struct{})
	}
	h.subscribers[taskID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if subscribers := h.subscribers[taskID]; subscribers != nil {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(h.subscribers, taskID)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(task *model.VideoGenerationTask) {
	view := ViewOf(task)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers[task.ID] {
		select {
		case ch <- view:
		default:
		}
	}
}

package service

import "ai-video/internal/repository"

// ListSortRequest is shared by paginated admin-list query DTOs. The binding
// allowlist prevents arbitrary column names or SQL fragments from reaching a
// repository.
type ListSortRequest struct {
	SortBy    string `form:"sort_by" binding:"omitempty,oneof=id sort"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

func (r ListSortRequest) listSort() repository.ListSort {
	return repository.ListSort{Field: r.SortBy, Order: r.SortOrder}
}

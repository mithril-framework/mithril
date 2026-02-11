package utils

import (
	"fmt"
	"math"
	"strconv"
)

// PaginationRequest represents pagination parameters from request
type PaginationRequest struct {
	Page     int    `json:"page"`      // Page number (1-based)
	PageSize int    `json:"page_size"` // Items per page
	SortBy   string `json:"sort_by"`   // Field to sort by
	SortDir  string `json:"sort_dir"`  // Sort direction (asc/desc)
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`        // Current page
	PageSize   int   `json:"page_size"`   // Items per page
	Total      int64 `json:"total"`       // Total number of items
	TotalPages int   `json:"total_pages"` // Total number of pages
	HasNext    bool  `json:"has_next"`    // Has next page
	HasPrev    bool  `json:"has_prev"`    // Has previous page
	NextPage   *int  `json:"next_page"`   // Next page number (null if no next page)
	PrevPage   *int  `json:"prev_page"`   // Previous page number (null if no prev page)
}

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Data interface{}    `json:"data"` // The actual data
	Meta PaginationMeta `json:"meta"` // Pagination metadata
}

// PaginationConfig holds configuration for pagination
type PaginationConfig struct {
	DefaultPageSize int    `json:"default_page_size"` // Default page size
	MaxPageSize     int    `json:"max_page_size"`     // Maximum allowed page size
	MinPageSize     int    `json:"min_page_size"`     // Minimum allowed page size
	DefaultSortBy   string `json:"default_sort_by"`   // Default sort field
	DefaultSortDir  string `json:"default_sort_dir"`  // Default sort direction
}

// DefaultPaginationConfig returns the default pagination configuration
func DefaultPaginationConfig() *PaginationConfig {
	return &PaginationConfig{
		DefaultPageSize: 10,
		MaxPageSize:     100,
		MinPageSize:     1,
		DefaultSortBy:   "id",
		DefaultSortDir:  "asc",
	}
}

// ParsePaginationRequest parses pagination parameters from query parameters
func ParsePaginationRequest(pageStr, pageSizeStr, sortBy, sortDir string, config *PaginationConfig) *PaginationRequest {
	if config == nil {
		config = DefaultPaginationConfig()
	}

	// Parse page
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse page size
	pageSize := config.DefaultPageSize
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil {
			// Enforce limits
			if ps > config.MaxPageSize {
				pageSize = config.MaxPageSize
			} else if ps < config.MinPageSize {
				pageSize = config.MinPageSize
			} else {
				pageSize = ps
			}
		}
	}

	// Parse sort field
	if sortBy == "" {
		sortBy = config.DefaultSortBy
	}

	// Parse sort direction
	if sortDir == "" {
		sortDir = config.DefaultSortDir
	}
	// Normalize sort direction
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	return &PaginationRequest{
		Page:     page,
		PageSize: pageSize,
		SortBy:   sortBy,
		SortDir:  sortDir,
	}
}

// CalculateOffset calculates the offset for database queries
func CalculateOffset(page, pageSize int) int {
	if page <= 0 {
		page = 1
	}
	return (page - 1) * pageSize
}

// CalculatePaginationMeta calculates pagination metadata
func CalculatePaginationMeta(page, pageSize int, total int64) PaginationMeta {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	var nextPage *int
	var prevPage *int

	if hasNext {
		next := page + 1
		nextPage = &next
	}

	if hasPrev {
		prev := page - 1
		prevPage = &prev
	}

	return PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		NextPage:   nextPage,
		PrevPage:   prevPage,
	}
}

// CreatePaginationResponse creates a paginated response
func CreatePaginationResponse(data interface{}, page, pageSize int, total int64) *PaginationResponse {
	meta := CalculatePaginationMeta(page, pageSize, total)

	return &PaginationResponse{
		Data: data,
		Meta: meta,
	}
}

// ValidatePaginationRequest validates pagination parameters
func ValidatePaginationRequest(req *PaginationRequest, config *PaginationConfig) error {
	if config == nil {
		config = DefaultPaginationConfig()
	}

	if req.Page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}

	if req.PageSize < config.MinPageSize {
		return fmt.Errorf("page size must be at least %d", config.MinPageSize)
	}

	if req.PageSize > config.MaxPageSize {
		return fmt.Errorf("page size must be at most %d", config.MaxPageSize)
	}

	if req.SortDir != "asc" && req.SortDir != "desc" {
		return fmt.Errorf("sort direction must be 'asc' or 'desc'")
	}

	return nil
}

// GetPaginationLinks generates pagination links for API responses
func GetPaginationLinks(meta PaginationMeta, baseURL string) map[string]interface{} {
	links := make(map[string]interface{})

	// Current page
	links["self"] = fmt.Sprintf("%s?page=%d&page_size=%d", baseURL, meta.Page, meta.PageSize)

	// First page
	links["first"] = fmt.Sprintf("%s?page=1&page_size=%d", baseURL, meta.PageSize)

	// Last page
	links["last"] = fmt.Sprintf("%s?page=%d&page_size=%d", baseURL, meta.TotalPages, meta.PageSize)

	// Next page
	if meta.HasNext && meta.NextPage != nil {
		links["next"] = fmt.Sprintf("%s?page=%d&page_size=%d", baseURL, *meta.NextPage, meta.PageSize)
	}

	// Previous page
	if meta.HasPrev && meta.PrevPage != nil {
		links["prev"] = fmt.Sprintf("%s?page=%d&page_size=%d", baseURL, *meta.PrevPage, meta.PageSize)
	}

	return links
}

// PaginationResponseWithLinks represents a paginated response with links
type PaginationResponseWithLinks struct {
	Data  interface{}            `json:"data"`
	Meta  PaginationMeta         `json:"meta"`
	Links map[string]interface{} `json:"links"`
}

// CreatePaginationResponseWithLinks creates a paginated response with navigation links
func CreatePaginationResponseWithLinks(data interface{}, page, pageSize int, total int64, baseURL string) *PaginationResponseWithLinks {
	meta := CalculatePaginationMeta(page, pageSize, total)
	links := GetPaginationLinks(meta, baseURL)

	return &PaginationResponseWithLinks{
		Data:  data,
		Meta:  meta,
		Links: links,
	}
}

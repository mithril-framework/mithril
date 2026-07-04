package utils

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	LastPage    int   `json:"last_page"`
	From        int   `json:"from"`
	To          int   `json:"to"`
	HasMore     bool  `json:"has_more"`
}

// PaginationLinks holds first/last/prev/next URLs.
type PaginationLinks struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
}

// PaginationResponse combines data with pagination and links.
type PaginationResponse struct {
	Data       interface{}     `json:"data"`
	Pagination PaginationMeta  `json:"pagination"`
	Links      PaginationLinks `json:"links"`
}

// NewPagination builds PaginationMeta for the given page, perPage, and total.
func NewPagination(page, perPage int, total int64) PaginationMeta {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage < 1 {
		lastPage = 1
	}
	from := (page-1)*perPage + 1
	to := page * perPage
	if to > int(total) {
		to = int(total)
	}
	return PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		LastPage:    lastPage,
		From:        from,
		To:          to,
		HasMore:     page < lastPage,
	}
}

// GeneratePaginationLinks builds First/Last/Prev/Next URLs for the given baseURL and meta.
func GeneratePaginationLinks(baseURL string, meta PaginationMeta) PaginationLinks {
	var links PaginationLinks
	links.First = fmt.Sprintf("%s?page=1&per_page=%d", baseURL, meta.PerPage)
	links.Last = fmt.Sprintf("%s?page=%d&per_page=%d", baseURL, meta.LastPage, meta.PerPage)
	if meta.CurrentPage > 1 {
		links.Prev = fmt.Sprintf("%s?page=%d&per_page=%d", baseURL, meta.CurrentPage-1, meta.PerPage)
	}
	if meta.HasMore {
		links.Next = fmt.Sprintf("%s?page=%d&per_page=%d", baseURL, meta.CurrentPage+1, meta.PerPage)
	}
	return links
}

// ParsePaginationParams reads page and per_page from Fiber query (defaults 1, 15; per_page capped at 100).
func ParsePaginationParams(c fiber.Ctx) (page, perPage int) {
	page, _ = strconv.Atoi(c.Query("page", "1"))
	perPage, _ = strconv.Atoi(c.Query("per_page", "15"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}
	return page, perPage
}

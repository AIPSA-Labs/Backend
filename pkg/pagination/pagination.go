package pagination

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type Pagination struct {
	Page       int
	PerPage    int
	Offset     int
	Total      int
	TotalPages int
}

func New(c fiber.Ctx) *Pagination {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	return &Pagination{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

func (p *Pagination) SetTotal(total int) {
	p.Total = total
	p.TotalPages = int(math.Ceil(float64(total) / float64(p.PerPage)))
}

func (p *Pagination) Meta() map[string]interface{} {
	return map[string]interface{}{
		"page":        p.Page,
		"per_page":    p.PerPage,
		"total":       p.Total,
		"total_pages": p.TotalPages,
	}
}

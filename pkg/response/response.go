package response

import (
	"github.com/gofiber/fiber/v3"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func Success(c fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success: true,
		Data:    data,
	})
}

func Created(c fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success: true,
		Data:    data,
	})
}

func Paginated(c fiber.Ctx, data interface{}, meta *Meta) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Error(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(Response{
		Success: false,
		Error:   message,
	})
}

func ErrorResponse(message string) Response {
	return Response{
		Success: false,
		Error:   message,
	}
}

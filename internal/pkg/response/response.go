package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API response structure.
type Response struct {
	Code    int         `json:"code" example:"0"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Code    int    `json:"code" example:"40000"`
	Message string `json:"message" example:"invalid request"`
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse struct {
	Code    int           `json:"code" example:"0"`
	Message string        `json:"message" example:"success"`
	Data    PaginatedData `json:"data"`
}

// PaginatedData contains pagination information.
type PaginatedData struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total" example:"100"`
	Page       int         `json:"page" example:"1"`
	PageSize   int         `json:"page_size" example:"20"`
	TotalPages int         `json:"total_pages" example:"5"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

func Paginated(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PaginatedData{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

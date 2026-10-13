package response

import "github.com/gin-gonic/gin"

type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Paging  *Pagination `json:"pagination,omitempty"`
}

type ErrorResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Errors  []ErrorItem  `json:"errors,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ErrorItem struct {
	Field  string `json:"field,omitempty"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func Success(c *gin.Context, status int, message string, data interface{}, pagination *Pagination) {
	c.JSON(status, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
		Paging:  pagination,
	})
}

func Error(c *gin.Context, status int, message string, errors []ErrorItem) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}

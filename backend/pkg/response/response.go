package response

import (
	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type successBody struct {
	Data any `json:"data"`
}

// JSON writes status + any value as JSON. Used for ad-hoc shapes.
func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

// Error writes a structured {code, message} error response.
func Error(c *gin.Context, status int, code, message string) {
	JSON(c, status, errorBody{Code: code, Message: message})
}

// Success wraps data in {data: ...} and writes the given status.
func Success(c *gin.Context, status int, data any) {
	JSON(c, status, successBody{Data: data})
}

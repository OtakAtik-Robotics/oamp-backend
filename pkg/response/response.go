package response

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Status: "success",
		Data:   data,
	})
}

func OKWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func CreatedWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Status:  "error",
		Message: message,
		Data:    nil,
	})
}

func FormatBindError(err error) string {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		var msgs []string
		for _, e := range validationErrs {
			msgs = append(msgs, formatFieldError(e))
		}
		return strings.Join(msgs, "; ")
	}
	return "Invalid request body"
}

func formatFieldError(e validator.FieldError) string {
	field := e.Field()
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "gte":
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	default:
		return fmt.Sprintf("%s failed validation", field)
	}
}

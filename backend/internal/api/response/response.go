package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
)

type Envelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, Envelope{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Fail(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Envelope{
		Code:    code,
		Message: message,
		Data:    gin.H{},
	})
}

func FromError(c *gin.Context, err error) {
	appErr, ok := apperror.As(err)
	if !ok {
		Fail(c, http.StatusInternalServerError, apperror.ErrInternal.Code, apperror.ErrInternal.Message)
		return
	}
	Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}

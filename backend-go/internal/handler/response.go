package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// fail 返回错误响应，前端读取 error.response.data.detail。
func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"detail": msg})
}

func badRequest(c *gin.Context, msg string) { fail(c, http.StatusBadRequest, msg) }
func unauthorized(c *gin.Context, msg string) {
	fail(c, http.StatusUnauthorized, msg)
}
func notFound(c *gin.Context, msg string)   { fail(c, http.StatusNotFound, msg) }
func serverError(c *gin.Context, msg string) {
	fail(c, http.StatusInternalServerError, msg)
}

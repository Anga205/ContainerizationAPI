package routes

import (
	"github.com/gin-gonic/gin"
)

func Setup(router *gin.Engine) {
	router.GET("/", root)
	router.POST("/v2/execute", CallDispatcher)
	router.POST("/execute", SimpleDispatcher)
}

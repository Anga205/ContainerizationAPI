package routes

import (
	"CodeSandboxAPI/dispatcher"

	"github.com/gin-gonic/gin"
)

func Setup(router *gin.Engine) {
	router.GET("/", root)
	router.POST("/execute", dispatcher.CallDispatcher)
}

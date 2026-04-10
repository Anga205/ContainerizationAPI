package routes

import (
	"CodeSandboxAPI/config"

	"github.com/gin-gonic/gin"
)

func Setup(router *gin.Engine) {
	router.GET("/", root)
	router.GET("/v2/execute", root)
	router.GET("/execute", root)
	router.POST("/v2/execute", CallDispatcher)
	router.POST("/execute", SimpleDispatcher)

	if config.Config.Globals.ENABLE_DEBUG_ROUTES {
		router.GET("/debug/config", debugConfig)
	}
}

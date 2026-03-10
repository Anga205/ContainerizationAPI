package routes

import "github.com/gin-gonic/gin"

func Setup(router *gin.Engine) {
	router.GET("/", root)
}

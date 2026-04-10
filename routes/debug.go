package routes

import (
	"CodeSandboxAPI/config"

	"github.com/gin-gonic/gin"
)

func debugConfig(c *gin.Context) {
	c.JSON(200, config.Config)
}

package routes

import (
	"CodeSandboxAPI/dispatcher"
	"CodeSandboxAPI/models"

	"github.com/gin-gonic/gin"
)

func CallDispatcher(c *gin.Context) {
	var req models.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	resp, err := dispatcher.Dispatch(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(200, resp)
}

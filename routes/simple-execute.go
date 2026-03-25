package routes

import (
	"CodeSandboxAPI/dispatcher"
	"CodeSandboxAPI/models"

	"github.com/gin-gonic/gin"
)

func SimpleDispatcher(c *gin.Context) {
	var req models.SimpleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	convertedReq := req.ToRequest()
	resp, err := dispatcher.Dispatch(convertedReq)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	convertedResp := resp.ToSimpleResponse()
	c.JSON(200, convertedResp)
}

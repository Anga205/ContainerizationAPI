package routes

import (
	"CodeSandboxAPI/dispatcher"
	"CodeSandboxAPI/models"
	"fmt"

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
	convertedResp := resp.ToSimpleResponse()
	if err != nil {
		fmt.Printf("Error during dispatch: %v\n", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(200, convertedResp)
}

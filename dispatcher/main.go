package dispatcher

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/models"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func CallDispatcher(c *gin.Context) {
	var req models.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	req.Timeout *= time.Second // Convert seconds to duration
	if req.Timeout == 0 {
		req.Timeout = config.Config.Limits.DefaultTimeout
	} else if req.Timeout > config.Config.Limits.MaxTimeout {
		req.Timeout = config.Config.Limits.MaxTimeout
	}

	if req.MemoryLimit == 0 {
		req.MemoryLimit = config.Config.Limits.DefaultMemoryLimit
	} else if req.MemoryLimit > config.Config.Limits.MaxMemoryLimit {
		req.MemoryLimit = config.Config.Limits.MaxMemoryLimit
	}

	resp, err := dispatch(req)
	if err != nil {
		fmt.Printf("Error during dispatch: %v\n", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(200, resp)
}

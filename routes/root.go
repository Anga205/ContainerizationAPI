package routes

import "github.com/gin-gonic/gin"

func root(c *gin.Context) {
	c.Redirect(301, "https://github.com/Anga205/CodeSandboxAPI")
}

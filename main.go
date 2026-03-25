package main

import (
	"CodeSandboxAPI/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	routes.Setup(router)
	router.Run()
}

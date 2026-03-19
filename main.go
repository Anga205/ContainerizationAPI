package main

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/routes"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Printf("Configuration loaded: %+v\n", config.Config)
	router := gin.Default()
	routes.Setup(router)
	router.Run()
}

package main

import (
	"CodeSandboxAPI/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.Use(cors.Default())

	routes.Setup(router)
	router.Run()
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getHome godoc
// @Summary Show the status of server.
// @Description get the status of server.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router / [get]
func getHome(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"message": "Hello World From MMA Backend API v1"})
}

// registerExampleRoutes registers the example '/helloworld' route
func registerExampleRoutes(group *gin.RouterGroup) {
	eg := group.Group("/example")
	eg.GET("/helloworld", getHome)
}

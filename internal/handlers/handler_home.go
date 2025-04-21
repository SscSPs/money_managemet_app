package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetHome godoc
// @Summary Show a welcome message
// @Description get a welcome message
// @Tags home
// @Accept  json
// @Produce  json
// @Success 200 {array} string "Welcome to my Go app!"
// @Router /example/helloworld [get]
func GetHome(c *gin.Context) {
	c.JSON(http.StatusOK, [1]string{"Welcome to my Go app!"})
}

package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorLog(c *gin.Context) {
	// print to log what the error was and who made it
	// read the JSON error object

	user, ok := getVerifyUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{})
	}

	var body struct {
		Text string `json:"text"`
	}

	// Bind JSON body to struct
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Log the error
	log.Printf("Error logged by user: %s Error details: %s", user.Username, body.Text)

	c.JSON(http.StatusOK, gin.H{"status": "error logged"})
}

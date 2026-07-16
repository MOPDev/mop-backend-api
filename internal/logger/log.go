package logger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/models"
	"github.com/gin-gonic/gin"
)

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

func Info(msg string) {
	fmt.Printf("%s | %sINFO%s  | %s\n",
		time.Now().Format("2006/01/02 15:04:05"),
		colorCyan, colorReset,
		msg,
	)
}

func Warn(msg string) {
	fmt.Printf("%s | %sWARN%s  | %s\n",
		time.Now().Format("2006/01/02 15:04:05"),
		colorYellow, colorReset,
		msg,
	)
}

func Error(msg string) {
	fmt.Printf("%s | %sERROR%s | %s\n",
		time.Now().Format("2006/01/02 15:04:05"),
		colorRed, colorReset,
		msg,
	)
	currentVal, err := json.Marshal(gin.H{"text": msg})
	if err != nil {
		return // ponytail: can't persist, console log above already happened
	}

	entry := models.ActivityLog{
		ActingUserID: 0,
		ActionType:   "ERROR LOGGER",
		CurrentVal:   currentVal,
	}
	initializers.DB.Create(&entry)
}

func Infof(format string, args ...any)  { Info(fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...any)  { Warn(fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...any) { Error(fmt.Sprintf(format, args...)) }

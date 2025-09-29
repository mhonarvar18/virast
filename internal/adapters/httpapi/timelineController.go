package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TimelineController struct{ tc TimelineUseCase }

func NewTimelineController(tc TimelineUseCase) *TimelineController {
	return &TimelineController{tc: tc}
}

func (ctrl *TimelineController) GetTimelineByUserID(c *gin.Context) {
	// گرفتن userID از context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	// گرفتن start و limit از Query params و مقداردهی پیش‌فرض
	startStr := c.DefaultQuery("start", "0")
	limitStr := c.DefaultQuery("limit", "20")

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start"})
		return
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	// فراخوانی سرویس GetTimeline
	timelinePosts, err := ctrl.tc.GetTimelineByUserID(c.Request.Context(), userID.(string), start, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch timeline"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"timeline": timelinePosts})
}

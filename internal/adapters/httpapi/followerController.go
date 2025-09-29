package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type FollowerController struct{ fc FollowerUseCase }

func NewFollowerController(fc FollowerUseCase) *FollowerController {
	return &FollowerController{fc: fc}
}

func (ctl *FollowerController) FollowUser(c *gin.Context) {
	var req struct {
		FollowedID string `json:"followed_id" binding:"required"`
	}

	// اعتبارسنجی JSON ورودی
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	// گرفتن userID از context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	// جلوگیری از دنبال کردن خود کاربر
	if userID.(string) == req.FollowedID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot follow yourself"})
		return
	}

	// اعتبارسنجی UUID
	if _, err := uuid.FromString(req.FollowedID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid followed_id"})
		return
	}

	// بررسی اینکه کاربر قبلاً دنبال شده یا نه
	isFollowing, err := ctl.fc.IsFollowing(c.Request.Context(), userID.(string), req.FollowedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not check follow status"})
		return
	}
	if isFollowing {
		c.JSON(http.StatusConflict, gin.H{"error": "already following this user"})
		return
	}

	// فراخوانی سرویس Follow
	if err := ctl.fc.FollowUser(c.Request.Context(), userID.(string), req.FollowedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not follow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully followed user"})
}

func (ctl *FollowerController) UnfollowUser(c *gin.Context) {
	var req struct {
		UnfollowedID string `json:"unfollowed_id" binding:"required"`
	}

	// اعتبارسنجی JSON ورودی
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	// گرفتن userID از context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	// بررسی اینکه کاربر قبلاً دنبال شده یا نه
	isFollowing, err := ctl.fc.IsFollowing(c.Request.Context(), userID.(string), req.UnfollowedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not check follow status"})
		return
	}
	if !isFollowing {
		c.JSON(http.StatusConflict, gin.H{"error": "you are not following this user"})
		return
	}

	// جلوگیری از آنفالو کردن خود
	if userID.(string) == req.UnfollowedID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot unfollow yourself"})
		return
	}

	// اعتبارسنجی UUID
	if _, err := uuid.FromString(req.UnfollowedID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unfollowed_id"})
		return
	}

	// فراخوانی سرویس Unfollow
	if err := ctl.fc.UnfollowUser(c.Request.Context(), userID.(string), req.UnfollowedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not unfollow user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully unfollowed user"})
}

func (ctl *FollowerController) GetFollowersByUserID(c *gin.Context) {
	// گرفتن userID از context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	followers, err := ctl.fc.GetFollowersByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get followers"})
		return
	}
	c.JSON(http.StatusOK, followers)
}

func (ctl *FollowerController) GetFollowingByUserID(c *gin.Context) {
	// گرفتن userID از context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	following, err := ctl.fc.GetFollowingByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get following"})
		return
	}
	c.JSON(http.StatusOK, following)
}

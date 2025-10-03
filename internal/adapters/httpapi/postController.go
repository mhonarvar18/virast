package httpapi

import (
	"net/http"
	"virast/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PostController struct{ pc PostUseCase }

func NewPostController(pc PostUseCase) *PostController { return &PostController{pc: pc} }

func (ctl *PostController) CreatePost(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	// گرفتن userID از context
	userID, exists := c.Get("userID")
	config.Logger.Info("UserID from context:", zap.String("userID", userID.(string)))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	res, err := ctl.pc.CreatePost(c.Request.Context(), req.Content, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create post"})
		return
	}
	c.JSON(http.StatusCreated, res)
}

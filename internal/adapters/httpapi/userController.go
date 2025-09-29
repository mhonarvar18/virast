package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct{ uc UserUseCase }

func NewUserController(uc UserUseCase) *UserController { return &UserController{uc: uc} }

func (ctl *UserController) LoginUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	res, err := ctl.uc.LoginUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (ctl *UserController) RegisterUser(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Family   string `json:"family" binding:"required"`
		Username string `json:"username" binding:"required"`
		Mobile   string `json:"mobile" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	u, err := ctl.uc.RegisterUser(c.Request.Context(), req.Name, req.Family, req.Username, req.Mobile, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username or mobile already taken"})
		return
	}
	c.JSON(http.StatusCreated, u)
}

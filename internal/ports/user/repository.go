package user

import "virast/internal/core/user"

// UserRepository پورت برای ذخیره‌سازی و بازیابی کاربران
type UserRepository interface {
	Create(user *user.User) (*user.User, error)
	FindByUsernameOrMobile(username, mobile string) (*user.User, error)
	FindByUsername(username string) (*user.User, error)
}

// DTOها برای UseCase
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}

type UserDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Mobile   string `json:"mobile"`
}
package post

import (
	"virast/internal/core/post"
	userPort "virast/internal/ports/user"
)

// PostRepository پورت برای ذخیره‌سازی و بازیابی پست‌ها
type PostRepository interface {
	Create(post *post.Post) (*post.Post, error)
	FindByID(id string) (*post.Post, error)
	FindByUserID(userID string) ([]*post.Post, error)
}

// DTOها برای UseCase
type PostDTO struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	UserID    string            `json:"user_id"`
	User      *userPort.UserDTO `json:"user,omitempty"`
	CreatedAt string            `json:"created_at"`
}

package timeline

import (
	"time"
	"virast/internal/core/post"
	"virast/internal/core/user"
	"github.com/gofrs/uuid"
)

type Timeline struct {
	ID        uuid.UUID  `gorm:"primary_key;type:char(36);default:uuid()"`
	UserID    uuid.UUID  `gorm:"type:char(36);not null"`
	PostID    uuid.UUID  `gorm:"type:char(36);not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	DeletedAt *time.Time `gorm:"index"`
	
	User      user.User  `gorm:"-"` // ارتباط با مدل User
	Post      post.Post  `gorm:"-"` // ارتباط با مدل Post
}

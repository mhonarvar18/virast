package post

import (
	"github.com/gofrs/uuid"
	"time"
	"virast/internal/core/user"
)

type Post struct {
	ID        uuid.UUID  `gorm:"primary_key;type:char(36);default:uuid()"`
	Content   string     `gorm:"type:text;not null"`
	UserID    uuid.UUID  `gorm:"type:char(36);not null"`
	User      user.User  `gorm:"foreignkey:UserID"` // ارتباط با مدل User
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"`
	DeletedAt *time.Time `gorm:"index"`
}

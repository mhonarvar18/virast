package timeline

import (
	"github.com/gofrs/uuid"
	"time"
	"virast/internal/core/post"
	"virast/internal/core/user"
)

type Timeline struct {
	ID        uuid.UUID  `gorm:"primary_key;type:char(36);default:uuid()"`
	UserID    uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:uniq_user_post"`
	PostID    uuid.UUID  `gorm:"type:char(36);not null;uniqueIndex:uniq_user_post"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	DeletedAt *time.Time `gorm:"index"`

	User user.User `gorm:"-"` // ارتباط با مدل User
	Post post.Post `gorm:"-"` // ارتباط با مدل Post
}

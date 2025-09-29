package fanoutqueue

import (
	"github.com/gofrs/uuid"
	"time"
	"virast/internal/core/post"
	"virast/internal/core/user"
)

type FanoutQueue struct {
	ID          uuid.UUID  `gorm:"primary_key;type:char(36);default:uuid()"`
	PostID      uuid.UUID  `gorm:"type:char(36);not null"`
	Post        post.Post  `gorm:"foreignkey:PostID;references:ID"`
	UserID      uuid.UUID  `gorm:"type:char(36);not null"`
	User        user.User  `gorm:"foreignKey:UserID;references:ID"`
	Status      string     `gorm:"type:varchar(20);not null"` // pending, done, failed
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	ProcessedAt *time.Time `gorm:"index"`
	DeletedAt   *time.Time `gorm:"index"`
}

package follower

import (
	"time"
	"virast/internal/core/user"
	"github.com/gofrs/uuid"
)

type Follower struct {
	ID         uuid.UUID  `gorm:"primary_key;type:char(36);default:uuid()"`
	UserID     uuid.UUID  `gorm:"type:char(36);not null"`
	User       user.User  `gorm:"foreignkey:UserID"` // ارتباط با مدل User
	FollowerID uuid.UUID  `gorm:"type:char(36);not null"`
	Follower   user.User  `gorm:"foreignkey:FollowerID"` // ارتباط با مدل Follower
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	DeletedAt  *time.Time `gorm:"index"`
}

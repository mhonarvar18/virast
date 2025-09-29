package user

import (
	"github.com/gofrs/uuid"
	"time"
)

type User struct {
	ID        uuid.UUID `gorm:"primary_key;type:char(36);default:uuid()"`
	Name      string    `gorm:"not null"`
	Family    string    `gorm:"not null"`
	Username  string    `gorm:"unique;not null"`
	Mobile    string    `gorm:"unique;not null"`
	Password  string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	DeletedAt *time.Time `gorm:"index"`
}

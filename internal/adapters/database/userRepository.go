package database

import (
	"virast/internal/config"
	"virast/internal/core/user"
)

// UserRepositoryDatabase پیاده‌سازی UserRepository برای دیتابیس
type UserRepositoryDatabase struct{}

// NewUserRepositoryDatabase سازنده UserRepositoryDatabase
func NewUserRepositoryDatabase() *UserRepositoryDatabase {
	return &UserRepositoryDatabase{}
}

func (repo *UserRepositoryDatabase) Create(user *user.User) (*user.User, error) {
	if err := config.DB.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *UserRepositoryDatabase) FindByUsernameOrMobile(username, mobile string) (*user.User, error) {
	var user user.User
	if err := config.DB.Where("username = ? OR mobile = ?", username, mobile).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *UserRepositoryDatabase) FindByUsername(username string) (*user.User, error) {
	var user user.User
	if err := config.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

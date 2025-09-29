package database

import (
	"virast/internal/config"
	"virast/internal/core/post"
)

// PostRepositoryDatabase پیاده‌سازی PostRepository برای دیتابیس
type PostRepositoryDatabase struct{}

// NewPostRepositoryDatabase سازنده PostRepositoryDatabase
func NewPostRepositoryDatabase() *PostRepositoryDatabase {
	return &PostRepositoryDatabase{}
}

func (repo *PostRepositoryDatabase) Create(post *post.Post) (*post.Post, error) {
	if err := config.DB.Create(post).Error; err != nil {
		return nil, err
	}
	return post, nil
}

func (repo *PostRepositoryDatabase) FindByID(id string) (*post.Post, error) {
	var post post.Post
	if err := config.DB.Where("id = ?", id).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (repo *PostRepositoryDatabase) FindByUserID(userID string) ([]*post.Post, error) {
	var posts []*post.Post
	if err := config.DB.Where("user_id = ?", userID).Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}
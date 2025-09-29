package httpapi

import (
	"context"
	"virast/internal/adapters/httpapi/middleware"
	followerPort "virast/internal/ports/follower"
	postPort "virast/internal/ports/post"
	userPort "virast/internal/ports/user"

	"github.com/gin-gonic/gin"
)

// UserUseCase: اینترفیسِ لازم برای کنترلر/روتر (Inbound Port)
type UserUseCase interface {
	LoginUser(ctx context.Context, username, password string) (*userPort.LoginResponse, error)
	RegisterUser(ctx context.Context, name, family, username, mobile, password string) (*userPort.UserDTO, error)
}

type PostUseCase interface {
	CreatePost(ctx context.Context, content, userID string) (*postPort.PostDTO, error)
}

type FollowerUseCase interface {
	FollowUser(ctx context.Context, followerID, followeeID string) error
	UnfollowUser(ctx context.Context, followerID, followeeID string) error
	GetFollowersByUserID(ctx context.Context, userID string) ([]*followerPort.FollowerDTO, error)
	GetFollowingByUserID(ctx context.Context, userID string) ([]*followerPort.FollowerDTO, error)
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
}

type TimelineUseCase interface {
	GetTimelineByUserID(ctx context.Context, userID string, start int64, limit int64) ([]*postPort.PostDTO, error)
}

// فقط روتینگ: UseCase از بیرون تزریق می‌شود
func SetupRoutes(
	userUC UserUseCase,
	postUC PostUseCase,
	followerUC FollowerUseCase,
	timelineUC TimelineUseCase,
) *gin.Engine {
	r := gin.Default()
	uc := NewUserController(userUC)
	pc := NewPostController(postUC)
	fc := NewFollowerController(followerUC)
	tc := NewTimelineController(timelineUC)

	// مسیرهای ثبت‌نام و ورود بدون JWT Middleware
	r.POST("/register", uc.RegisterUser)
	r.POST("/login", uc.LoginUser)

	// مسیر ایجاد پست با JWT Middleware
	r.POST("/post", middleware.JWTAuthMiddleware(), pc.CreatePost)

	// مسیرهای دنبال کردن و دریافت دنبال‌کنندگان با JWT Middleware
	r.POST("/follow", middleware.JWTAuthMiddleware(), fc.FollowUser)
	r.POST("/unfollow", middleware.JWTAuthMiddleware(), fc.UnfollowUser)
	r.GET("/followers", middleware.JWTAuthMiddleware(), fc.GetFollowersByUserID)
	r.GET("/following", middleware.JWTAuthMiddleware(), fc.GetFollowingByUserID)

	//
	r.GET("/timeline", middleware.JWTAuthMiddleware(), tc.GetTimelineByUserID)
	return r
}

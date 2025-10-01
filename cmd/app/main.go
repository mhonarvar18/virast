package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	dbadapter "virast/internal/adapters/database"
	"virast/internal/adapters/httpapi"
	redisadapter "virast/internal/adapters/redis"
	"virast/internal/config"
	"virast/internal/core/fanoutqueue"
	"virast/internal/core/follower"
	followerapp "virast/internal/core/follower/service"
	"virast/internal/core/post"
	postapp "virast/internal/core/post/service"
	"virast/internal/core/timeline"
	timelineapp "virast/internal/core/timeline/service"
	"virast/internal/core/user"
	userapp "virast/internal/core/user/service"
	"virast/internal/workers"

	"go.uber.org/zap"
)

func main() {
	config.InitLogger()
	config.Init() // بارگذاری تنظیمات از .env

	// اتصال به دیتابیس و اجرای مایگریشن‌ها
	config.InitDB()

	// اعمال مایگریشن برای مدل‌ها
	if err := config.DB.AutoMigrate(
		&user.User{},
		&post.Post{},
		&follower.Follower{},
		&timeline.Timeline{},
		&fanoutqueue.FanoutQueue{},
	); err != nil {
		config.Logger.Fatal("Error during migrations:", zap.Error(err))
	}

	config.Logger.Info("✅ Database migrations completed")

	// اتصال به Redis
	config.InitRedis()

	// بستن منابع بعد از اتمام کار سرور
	defer closeResources(config.Logger)

	// چاپ پیغام قبل از راه‌اندازی سرور
	config.Logger.Info("App is running...")

	userRepo := dbadapter.NewUserRepositoryDatabase()                                                // آداپتر خروجی
	postRepo := dbadapter.NewPostRepositoryDatabase()                                                // آداپتر خروجی
	fanoutRedis := redisadapter.NewFanoutRepositoryRedis(config.RedisClient)                         // آداپتر خروجی
	fanoutRepo := dbadapter.NewFanoutRepositoryDatabase()                                            // آداپتر خروجی
	followerRepo := dbadapter.NewFollowerRepositoryDatabase()                                        // آداپتر خروجی
	timelineRepo := dbadapter.NewtimelineRepositoryDatabase()                                        // آداپتر خروجی
	userSvc := userapp.NewUserService(userRepo, []byte(os.Getenv("JWT_SECRET")))                     // یوزکیس/سرویس
	postSvc := postapp.NewPostService(postRepo, fanoutRepo, fanoutRedis, followerRepo, timelineRepo) // یوزکیس/سرویس
	followerScv := followerapp.NewFollowerService(followerRepo)                                      // یوزکیس/سرویس
	timelineScv := timelineapp.NewTimelineService(timelineRepo)                                      // یوزکیس/سرویس
	r := httpapi.SetupRoutes(userSvc, postSvc, followerScv, timelineScv)                             // تزریق یوزکیس به آداپتر ورودی
	// -------------------------------------------

	batchSizeStr := os.Getenv("BATCH_SIZE") // تعداد رکوردهای batch برای Redis و timeline
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil || batchSize <= 0 {
		batchSize = 100 // مقدار پیش‌فرض
	}

	if err != nil {
		config.Logger.Fatal("Failed to initialize logger:", zap.Error(err))
	}

	fanoutWorker := workers.NewFanoutWorker(fanoutRepo, fanoutRedis, followerRepo, timelineRepo, batchSize, config.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TEST
	testStability(ctx, config.Logger, userSvc, postSvc, followerScv)
	// End TEST

	// اجرای worker در پس‌زمینه
	go fanoutWorker.Run(ctx)

	// اجرای سرور Gin (در اینجا سرور به صورت بلوکینگ عمل می‌کند)
	if err := r.Run(":" + os.Getenv("APP_PORT")); err != nil {
		config.Logger.Fatal("Server failed to start:", zap.Error(err))
	}
}

// closeResources بستن اتصالات به Redis و دیتابیس
func closeResources(logger *zap.Logger) {
	// بستن اتصال به Redis
	if err := config.RedisClient.Close(); err != nil {
		logger.Error("Error closing Redis connection:", zap.Error(err))
	}

	// بستن اتصال دیتابیس
	sqlDB, err := config.DB.DB() // گرفتن *sql.DB از *gorm.DB
	if err != nil {
		logger.Error("Error getting raw DB:", zap.Error(err))
		return
	}

	if err := sqlDB.Close(); err != nil {
		logger.Error("Error closing database connection:", zap.Error(err))
	}
}

func testStability(ctx context.Context, logger *zap.Logger, userSvc *userapp.UserService, postSvc *postapp.PostService, followerSvc *followerapp.FollowerService) {
	const numUsers = 500
	const postsPerUser = 10

	logger.Info("🚀 Starting testStability: creating users...")

	// 1️⃣ ساخت کاربران
	userIDs := make([]string, 0, numUsers)
	for i := 0; i < numUsers; i++ {
		username := fmt.Sprintf("testuser%d", i)
		mobile := fmt.Sprintf("0912%07d", i)
		u, err := userSvc.RegisterUser(ctx, "Test"+strconv.Itoa(i), "User", username, mobile, "password")
		if err != nil {
			logger.Error("❌ Error creating user", zap.String("username", username), zap.Error(err))
			continue
		}
		userIDs = append(userIDs, u.ID)
		if (i+1)%50 == 0 {
			logger.Info("✅ Created users so far", zap.Int("count", i+1))
		}
	}

	logger.Info("✅ Finished creating users", zap.Int("count", len(userIDs)))

	// 2️⃣ همه کاربرا همدیگه رو فالو کنن
	logger.Info("🚀 Starting follow setup...")
	count := 0
	for _, followerID := range userIDs {
		for _, followeeID := range userIDs {
			if followerID == followeeID {
				continue // جلوگیری از self-follow
			}
			err := followerSvc.FollowUser(ctx, followerID, followeeID)
			if err != nil {
				logger.Error("❌ Error: user could not follow", zap.String("followerID", followerID), zap.String("followeeID", followeeID), zap.Error(err))
				continue
			}
			count++
			if count%1000 == 0 {
				logger.Info("➡️ Processed follow relationships", zap.Int("count", count))
			}
		}
	}
	logger.Info("✅ Follow setup completed", zap.Int("count", count))

	// 3️⃣ هر کاربر ۱۰ پست ثبت کنه
	logger.Info("🚀 Starting post creation...")
	postCount := 0
	for _, uid := range userIDs {
		for p := 1; p <= postsPerUser; p++ {
			content := fmt.Sprintf("Post %d by user %s", p, uid)
			postDTO, err := postSvc.CreatePost(ctx, content, uid)
			if err != nil {
				logger.Error("❌ Error creating post", zap.String("userID", uid), zap.Error(err))
				continue
			}
			postCount++
			if postCount%100 == 0 {
				logger.Info("➡️ Created posts so far", zap.Int("count", postCount))
			}
			logger.Info("📝 Created post", zap.String("ID", postDTO.ID), zap.String("Content", postDTO.Content), zap.String("UserID", uid))
		}
	}

	logger.Info("✅ Test data creation completed", zap.Int("count", postCount))
}

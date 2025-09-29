package main

import (
	"context"
	"fmt"
	"log"
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
)

func main() {
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
		log.Fatal("Error during migrations:", err)
	}

	log.Println("✅ Database migrations completed")

	// اتصال به Redis
	config.InitRedis()

	// بستن منابع بعد از اتمام کار سرور
	defer closeResources()

	// چاپ پیغام قبل از راه‌اندازی سرور
	log.Println("App is running...")

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
	fanoutWorker := workers.NewFanoutWorker(fanoutRepo, fanoutRedis, followerRepo, timelineRepo, batchSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TEST
	testStability(ctx, userSvc, postSvc, followerScv)
	// End TEST

	// اجرای worker در پس‌زمینه
	go fanoutWorker.Run(ctx)

	// اجرای سرور Gin (در اینجا سرور به صورت بلوکینگ عمل می‌کند)
	if err := r.Run(":" + os.Getenv("APP_PORT")); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}

// closeResources بستن اتصالات به Redis و دیتابیس
func closeResources() {
	// بستن اتصال به Redis
	if err := config.RedisClient.Close(); err != nil {
		log.Println("Error closing Redis connection:", err)
	}

	// بستن اتصال دیتابیس
	sqlDB, err := config.DB.DB() // گرفتن *sql.DB از *gorm.DB
	if err != nil {
		log.Println("Error getting raw DB:", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Println("Error closing database connection:", err)
	}
}

func testStability(ctx context.Context, userSvc *userapp.UserService, postSvc *postapp.PostService, followerSvc *followerapp.FollowerService) {
	const numUsers = 500
	const postsPerUser = 10

	fmt.Println("🚀 Starting testStability: creating users...")

	// 1️⃣ ساخت کاربران
	userIDs := make([]string, 0, numUsers)
	for i := 0; i < numUsers; i++ {
		username := fmt.Sprintf("testuser%d", i)
		mobile := fmt.Sprintf("0912%07d", i)
		u, err := userSvc.RegisterUser(ctx, "Test"+strconv.Itoa(i), "User", username, mobile, "password")
		if err != nil {
			log.Printf("❌ Error creating user %s: %v\n", username, err)
			continue
		}
		userIDs = append(userIDs, u.ID)
		if (i+1)%50 == 0 {
			fmt.Printf("✅ Created %d users so far\n", i+1)
		}
	}

	fmt.Printf("✅ Finished creating %d users\n", len(userIDs))

	// 2️⃣ همه کاربرا همدیگه رو فالو کنن
	fmt.Println("🚀 Starting follow setup...")
	count := 0
	for _, followerID := range userIDs {
		for _, followeeID := range userIDs {
			if followerID == followeeID {
				continue // جلوگیری از self-follow
			}
			err := followerSvc.FollowUser(ctx, followerID, followeeID)
			if err != nil {
				log.Printf("❌ Error: user %s could not follow %s: %v\n", followerID, followeeID, err)
				continue
			}
			count++
			if count%1000 == 0 {
				fmt.Printf("➡️ Processed %d follow relationships\n", count)
			}
		}
	}
	fmt.Printf("✅ Follow setup completed: total %d follow relationships created\n", count)

	// 3️⃣ هر کاربر ۱۰ پست ثبت کنه
	fmt.Println("🚀 Starting post creation...")
	postCount := 0
	for _, uid := range userIDs {
		for p := 1; p <= postsPerUser; p++ {
			content := fmt.Sprintf("Post %d by user %s", p, uid)
			postDTO, err := postSvc.CreatePost(ctx, content, uid)
			if err != nil {
				log.Printf("❌ Error creating post for user %s: %v\n", uid, err)
				continue
			}
			postCount++
			if postCount%100 == 0 {
				fmt.Printf("➡️ Created %d posts so far\n", postCount)
			}
			fmt.Printf("📝 Created post: ID=%s, Content='%s', UserID=%s\n", postDTO.ID, postDTO.Content, uid)
		}
	}

	fmt.Printf("✅ Test data creation completed: total %d posts created\n", postCount)
}

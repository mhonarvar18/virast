package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
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
	"golang.org/x/sync/errgroup"
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

	concurrency := 32 // تعداد goroutine های همزمان
	fanoutWorker := workers.NewFanoutWorker(fanoutRepo, fanoutRedis, followerRepo, timelineRepo, batchSize, concurrency, config.Logger)

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

	logger.Info("🚀 Starting testStability: creating users...")

	const numUsers = 1000
	const postsPerUser = 10

	const userConc = 16   // با pool DB هماهنگ کن
	const followConc = 32 // سبک‌تر/نوشتنی‌تر؟ بالاتر هم می‌تونی ولی مراقب لاک‌های DB باش
	const postConc = 32

	logger.Info("🚀 creating users...")

	userIDs, _ := createUsersConcurrent(ctx, logger, userSvc, numUsers, userConc)
	logger.Info("✅ created %d users", zap.Int("Count", len(userIDs)))
	logger.Info("✅ created %d users")

	logger.Info("🚀 creating follows (complete graph, no self)...")
	createFollowsWithPool(ctx, logger, followerSvc, userIDs, followConc)
	logger.Info("✅ follows done")

	logger.Info("🚀 creating posts...")
	createPostsConcurrent(ctx, logger, postSvc, userIDs, postsPerUser, postConc)
	logger.Info("✅ posts done")
}

func createUsersConcurrent(ctx context.Context, logger *zap.Logger, userSvc *userapp.UserService, numUsers, concurrency int) ([]string, error) {
	userIDs := make([]string, 0, numUsers)

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	eg, ctx := errgroup.WithContext(ctx)

	for i := 0; i < numUsers; i++ {
		i := i
		sem <- struct{}{}
		eg.Go(func() error {
			defer func() { <-sem }()

			username := fmt.Sprintf("testuser%d", i)
			mobile := fmt.Sprintf("0912%07d", i)
			u, err := userSvc.RegisterUser(ctx, "Test"+strconv.Itoa(i), "User", username, mobile, "password")
			if err != nil {
				logger.Error("❌ Error creating user", zap.String("username", username), zap.Error(err))
				return nil // ادامه بده؛ شکست یک مورد، کل کار رو متوقف نکنه
			}
			mu.Lock()
			userIDs = append(userIDs, u.ID)
			mu.Unlock()

			if (i+1)%50 == 0 {
				logger.Info("✅ Created users so far", zap.Int("count", i+1))
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return userIDs, err
	}
	return userIDs, nil
}

type followJob struct {
	followerID string
	followeeID string
}

func createFollowsWithPool(ctx context.Context, logger *zap.Logger, followerSvc *followerapp.FollowerService, userIDs []string, concurrency int) {
	jobs := make(chan followJob, concurrency*2)
	var wg sync.WaitGroup

	// Workers
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					if job.followerID == job.followeeID {
						continue
					}
					if err := followerSvc.FollowUser(ctx, job.followerID, job.followeeID); err != nil {
						// idempotent باش: اگر unique violation می‌ده، نادیده بگیر
						logger.Error("❌ Error: user could not follow", zap.String("followerID", job.followerID), zap.String("followeeID", job.followeeID), zap.Error(err))
					}
					// در صورت نیاز لاگ سبک
					logger.Info("➡️ Processed follow", zap.String("followerID", job.followerID), zap.String("followeeID", job.followeeID))
				}

			}
		}(w)

	}

	// Producer
	go func() {
		for _, followerID := range userIDs {
			for _, followeeID := range userIDs {
				if followerID == followeeID {
					continue
				}
				select {
				case <-ctx.Done():
					close(jobs)
					return
				case jobs <- followJob{followerID, followeeID}:
				}
			}
		}
		close(jobs)
	}()

	wg.Wait()
}

func createPostsConcurrent(ctx context.Context, logger *zap.Logger, postSvc *postapp.PostService, userIDs []string, postsPerUser, concurrency int) {
	sem := make(chan struct{}, concurrency)
	var eg errgroup.Group

	for _, uid := range userIDs {
		uid := uid
		for p := 1; p <= postsPerUser; p++ {
			p := p
			sem <- struct{}{}
			eg.Go(func() error {
				defer func() { <-sem }()
				content := fmt.Sprintf("Post %d by user %s", p, uid)
				postDTO, err := postSvc.CreatePost(ctx, content, uid)
				if err != nil {
					logger.Error("❌ Error creating post", zap.String("userID", uid), zap.Error(err))
					return nil
				}
				// در صورت نیاز لاگ سبک
				logger.Info("📝 Created post", zap.String("ID", postDTO.ID), zap.String("Content", postDTO.Content), zap.String("UserID", uid))
				return nil
			})
		}
	}
	_ = eg.Wait()
}

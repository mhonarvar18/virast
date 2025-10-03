package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
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
	concurrency := 32 // تعداد goroutine های همزمان
	fanoutWorker := workers.NewFanoutWorker(fanoutRepo, fanoutRedis, followerRepo, timelineRepo, batchSize, concurrency)

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
	const numUsers = 1000
	const postsPerUser = 10

	const userConc = 16   // با pool DB هماهنگ کن
	const followConc = 32 // سبک‌تر/نوشتنی‌تر؟ بالاتر هم می‌تونی ولی مراقب لاک‌های DB باش
	const postConc = 32

	log.Println("🚀 creating users...")
	userIDs, _ := createUsersConcurrent(ctx, userSvc, numUsers, userConc)
	log.Printf("✅ created %d users", len(userIDs))

	log.Println("🚀 creating follows (complete graph, no self)...")
	createFollowsWithPool(ctx, followerSvc, userIDs, followConc)
	log.Println("✅ follows done")

	log.Println("🚀 creating posts...")
	createPostsConcurrent(ctx, postSvc, userIDs, postsPerUser, postConc)
	log.Println("✅ posts done")
}

func createUsersConcurrent(ctx context.Context, userSvc *userapp.UserService, numUsers, concurrency int) ([]string, error) {
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
				log.Printf("❌ create user %s: %v", username, err)
				return nil // ادامه بده؛ شکست یک مورد، کل کار رو متوقف نکنه
			}
			mu.Lock()
			userIDs = append(userIDs, u.ID)
			mu.Unlock()

			if (i+1)%50 == 0 {
				log.Printf("✅ Created %d users so far", i+1)
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

func createFollowsWithPool(ctx context.Context, followerSvc *followerapp.FollowerService, userIDs []string, concurrency int) {
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
						log.Printf("⚠️ follow %s -> %s: %v", job.followerID, job.followeeID, err)
					}
					// در صورت نیاز لاگ سبک
					log.Printf("✅ %s followed %s", job.followerID, job.followeeID)
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

func createPostsConcurrent(ctx context.Context, postSvc *postapp.PostService, userIDs []string, postsPerUser, concurrency int) {
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
					log.Printf("❌ create post for user %s: %v", uid, err)
					return nil
				}
				// در صورت نیاز لاگ سبک
				log.Printf("📝 post=%s user=%s", postDTO.ID, uid)
				return nil
			})
		}
	}
	_ = eg.Wait()
}

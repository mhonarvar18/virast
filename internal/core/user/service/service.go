package userapp

import (
	"context"
	"errors"
	"log"
	"time"
	userEntity "virast/internal/core/user"
	userPort "virast/internal/ports/user"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserService سرویس مدیریت کاربران
type UserService struct {
	UserRepository userPort.UserRepository
	jwtKey         []byte
}

func NewUserService(repo userPort.UserRepository, jwtKey []byte) *UserService {
	return &UserService{
		UserRepository: repo,
		jwtKey:         jwtKey,
	}
}

// تعریف کلید JWT برای امضاء
var jwtKey = []byte("secret_key")

// LoginUser ورود کاربر و صدور توکن JWT
func (s *UserService) LoginUser(ctx context.Context, username string, password string) (*userPort.LoginResponse, error) {
	// پیدا کردن کاربر با یوزرنیم
	user, err := s.UserRepository.FindByUsername(username)
	if err != nil {
		log.Println("Error finding user:", err)
		return nil, errors.New("invalid credentials")
	}

	// مقایسه پسورد هش‌شده
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Println("invalid password")
		return nil, errors.New("invalid credentials")
	}

	// ایجاد JWT Token
	token, err := generateJWT(user)
	if err != nil {
		log.Println("Error generating JWT:", err)
		return nil, errors.New("could not generate token")
	}

	// بازگشت توکن
	return &userPort.LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// generateJWT برای تولید توکن JWT
func generateJWT(user *userEntity.User) (string, error) {
	// ایجاد اطلاعات توکن
	claims := &jwt.StandardClaims{
		Subject:   user.ID.String(),
		Issuer:    "virast",
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), // انقضای توکن بعد از 24 ساعت
	}

	// ایجاد توکن
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// امضاء توکن
	return token.SignedString(jwtKey)
}

// RegisterUser ثبت‌نام کاربر جدید
func (s *UserService) RegisterUser(ctx context.Context, name, family, username, mobile, password string) (*userPort.UserDTO, error) {
	// بررسی اینکه آیا کاربر با این یوزرنیم یا موبایل قبلاً ثبت شده است
	existingUser, err := s.UserRepository.FindByUsernameOrMobile(username, mobile)
	if err == nil && existingUser != nil {
		return nil, errors.New("username or mobile already taken")
	}

	// هش کردن پسورد
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// ایجاد کاربر جدید
	user := &userEntity.User{
		ID:       uuid.Must(uuid.NewV4()),
		Name:     name,
		Family:   family,
		Username: username,
		Mobile:   mobile,
		Password: string(hashedPassword),
	}

	// ذخیره کاربر در دیتابیس
	u, err := s.UserRepository.Create(user)
    if err != nil {
        return nil, err
    }

    return &userPort.UserDTO{
        ID:       u.ID.String(),
        Username: u.Username,
        Mobile:   u.Mobile,
    }, nil
}

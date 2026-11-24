package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"consiliumous-backend/internal/auth"
	"consiliumous-backend/internal/model"
)

// AuthService 处理注册、登录与当前用户查询。
type AuthService struct {
	DB        *gorm.DB
	JWTSecret string
}

func NewAuthService(db *gorm.DB, secret string) *AuthService {
	return &AuthService{DB: db, JWTSecret: secret}
}

type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register 创建新用户。邮箱唯一。
func (s *AuthService) Register(in RegisterInput) (*model.User, error) {
	var count int64
	if err := s.DB.Model(&model.User{}).Where("email = ?", in.Email).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("该邮箱已被注册")
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := model.User{Email: in.Email, Name: in.Name, PasswordHash: hash}
	if err := s.DB.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Login 校验邮箱密码，成功返回 JWT 与用户信息。
func (s *AuthService) Login(in LoginInput) (string, *model.User, error) {
	var u model.User
	if err := s.DB.Where("email = ?", in.Email).First(&u).Error; err != nil {
		return "", nil, errors.New("邮箱或密码错误")
	}
	if !auth.CheckPassword(u.PasswordHash, in.Password) {
		return "", nil, errors.New("邮箱或密码错误")
	}
	token, err := auth.GenerateToken(s.JWTSecret, u.ID, auth.DefaultTTL)
	if err != nil {
		return "", nil, err
	}
	return token, &u, nil
}

// GetByID 按 ID 查用户。
func (s *AuthService) GetByID(id uint) (*model.User, error) {
	var u model.User
	if err := s.DB.First(&u, id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &u, nil
}

// Search 按姓名或邮箱模糊搜索用户（US-PROJ-03：管理员搜索用户加入项目）。
func (s *AuthService) Search(q string) ([]model.User, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []model.User{}, nil
	}
	var users []model.User
	like := "%" + q + "%"
	err := s.DB.Where("name LIKE ? OR email LIKE ?", like, like).Order("id").Limit(20).Find(&users).Error
	return users, err
}

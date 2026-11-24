package service

import (
	"errors"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// PreferenceService 承载用户偏好（键值对）。用于虚拟项目计划组的折叠/排序等，仅对本人有效。
type PreferenceService struct {
	DB *gorm.DB
}

func NewPreferenceService(db *gorm.DB) *PreferenceService {
	return &PreferenceService{DB: db}
}

// GetAll 返回某用户的全部偏好（map[key]value）。
func (s *PreferenceService) GetAll(userID uint) (map[string]string, error) {
	var prefs []model.UserPreference
	if err := s.DB.Where("user_id = ?", userID).Find(&prefs).Error; err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, p := range prefs {
		m[p.Key] = p.Value
	}
	return m, nil
}

// Set 设置或更新某个偏好键。
func (s *PreferenceService) Set(userID uint, key, value string) error {
	var p model.UserPreference
	err := s.DB.Where("user_id = ? AND key = ?", userID, key).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.DB.Create(&model.UserPreference{UserID: userID, Key: key, Value: value}).Error
	}
	if err != nil {
		return err
	}
	p.Value = value
	return s.DB.Save(&p).Error
}

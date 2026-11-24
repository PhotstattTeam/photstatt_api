package database

import (
	"errors"

	"gorm.io/gorm"

	"consiliumous-backend/internal/auth"
	"consiliumous-backend/internal/model"
)

// SeedSystemAdmin 在启动时按环境变量指定系统管理员（PRD 3.2：系统初始化时指定，全局唯一角色）。
// 若该邮箱用户已存在则提升为系统管理员；若不存在且提供了初始密码则自动创建。
func SeedSystemAdmin(db *gorm.DB, email, password string) error {
	if email == "" {
		return nil
	}
	var u model.User
	err := db.Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if password == "" {
			return nil // 用户尚未注册且未提供密码：等其注册后下次启动再提升
		}
		hash, herr := auth.HashPassword(password)
		if herr != nil {
			return herr
		}
		u = model.User{Email: email, Name: email, PasswordHash: hash, IsSystemAdmin: true}
		return db.Create(&u).Error
	}
	if err != nil {
		return err
	}
	if !u.IsSystemAdmin {
		return db.Model(&u).Update("is_system_admin", true).Error
	}
	return nil
}

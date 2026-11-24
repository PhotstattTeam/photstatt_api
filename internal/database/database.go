package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"consiliumous-backend/internal/model"
)

// Open 建立数据库连接。
// 当前使用 SQLite（单文件，零安装）。切换 PostgreSQL 时：
//
//	go get gorm.io/driver/postgres
//	return gorm.Open(postgres.Open(dsn), ...)
func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
}

// Migrate 根据 model 中的结构体自动建表（首次运行 / 结构变更时）。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.SubProject{},
		&model.ProjectMember{},
		&model.Issue{},
		&model.Plan{},
		&model.Task{},
		&model.PlanGroup{},
		&model.AuditLog{},
		&model.Comment{},
		&model.UserPreference{},
	)
}

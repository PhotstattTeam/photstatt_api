package main

import (
	"log"

	"consiliumous-backend/internal/config"
	"consiliumous-backend/internal/database"
	"consiliumous-backend/internal/handler"
	"consiliumous-backend/internal/router"
	"consiliumous-backend/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("建表失败: %v", err)
	}
	if err := database.SeedSystemAdmin(db, cfg.SystemAdminEmail, cfg.SystemAdminPassword); err != nil {
		log.Fatalf("初始化系统管理员失败: %v", err)
	}

	authSvc := service.NewAuthService(db, cfg.JWTSecret)
	projSvc := service.NewProjectService(db)
	subSvc := service.NewSubProjectService(db)
	issueSvc := service.NewIssueService(db)
	planSvc := service.NewPlanService(db)
	taskSvc := service.NewTaskService(db)
	personalSvc := service.NewPersonalPlanService(db)
	syncSvc := service.NewSyncService(db)
	prefsSvc := service.NewPreferenceService(db)
	h := handler.NewHandler(authSvc, projSvc, subSvc, issueSvc, planSvc, taskSvc, personalSvc, syncSvc, prefsSvc)

	r := router.New(h, cfg)
	log.Printf("ConsiliumOus 后端启动，监听 :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

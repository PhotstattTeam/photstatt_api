package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 汇总全部运行时配置。
type Config struct {
	Port                string   // 监听端口
	DBPath              string   // SQLite 文件路径
	JWTSecret           string   // JWT 签名密钥
	CorsAllowedOrigins  []string // 跨域白名单
	SystemAdminEmail    string   // 系统管理员邮箱（启动时提升）
	SystemAdminPassword string   // 系统管理员初始密码（可选，仅首次创建时用）
}

// Load 从环境变量 / .env 文件读取配置。
func Load() *Config {
	_ = godotenv.Load() // 找不到 .env 也不报错，使用默认值

	return &Config{
		Port:                getenv("PORT", "8081"),
		DBPath:              getenv("DB_PATH", "consiliumous.db"),
		JWTSecret:           getenv("JWT_SECRET", "dev-secret-change-me"),
		CorsAllowedOrigins:  strings.Split(getenv("CORS_ALLOWED_ORIGINS", "*"), ","),
		SystemAdminEmail:    getenv("SYSTEM_ADMIN_EMAIL", ""),
		SystemAdminPassword: getenv("SYSTEM_ADMIN_PASSWORD", ""),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

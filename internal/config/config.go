package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 汇总应用运行所需的全部配置，统一从环境变量读取。
type Config struct {
	AppName          string
	Env              string
	Port             string
	GinMode          string
	CORSAllowOrigins []string
	DB               DBConfig
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	TimeZone        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Load 读取 .env（存在时）与环境变量，缺失项使用默认值。
func Load() (*Config, error) {
	// 没有 .env 时忽略：容器/K8s 里通常直接注入环境变量
	_ = godotenv.Load()

	cfg := &Config{
		AppName:          env("APP_NAME", "photstatt_api"),
		Env:              env("APP_ENV", "development"),
		Port:             env("APP_PORT", "8080"),
		GinMode:          env("GIN_MODE", "debug"),
		CORSAllowOrigins: splitAndTrim(env("CORS_ALLOW_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		DB: DBConfig{
			Host:     env("DB_HOST", "127.0.0.1"),
			Port:     env("DB_PORT", "5432"),
			User:     env("DB_USER", "photstatt"),
			Password: env("DB_PASSWORD", ""),
			Name:     env("DB_NAME", "photstatt"),
			SSLMode:  env("DB_SSLMODE", "disable"),
			TimeZone: env("DB_TIMEZONE", "Asia/Shanghai"),
		},
	}

	var err error
	if cfg.DB.MaxOpenConns, err = envInt("DB_MAX_OPEN_CONNS", 25); err != nil {
		return nil, err
	}
	if cfg.DB.MaxIdleConns, err = envInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		return nil, err
	}
	if cfg.DB.ConnMaxLifetime, err = envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) validate() error {
	if c.DB.Host == "" {
		return fmt.Errorf("config: DB_HOST 不能为空")
	}
	if c.DB.User == "" {
		return fmt.Errorf("config: DB_USER 不能为空")
	}
	if c.DB.Name == "" {
		return fmt.Errorf("config: DB_NAME 不能为空")
	}
	if c.DB.Password == "" {
		return fmt.Errorf("config: DB_PASSWORD 未设置（请复制 .env.example 为 .env）")
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("config: APP_PORT %q 不是合法端口", c.Port)
	}
	if _, err := strconv.Atoi(c.DB.Port); err != nil {
		return fmt.Errorf("config: DB_PORT %q 不是合法端口", c.DB.Port)
	}
	if strings.ContainsAny(c.DB.TimeZone, " '\"\\\t\n") {
		return fmt.Errorf("config: DB_TIMEZONE %q 含非法字符", c.DB.TimeZone)
	}
	if c.DB.MaxIdleConns > c.DB.MaxOpenConns {
		return fmt.Errorf("config: DB_MAX_IDLE_CONNS(%d) 不应大于 DB_MAX_OPEN_CONNS(%d)",
			c.DB.MaxIdleConns, c.DB.MaxOpenConns)
	}
	switch c.GinMode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("config: GIN_MODE 只能是 debug/release/test，当前为 %q", c.GinMode)
	}
	return nil
}

// DSN 组装 key/value 形式的 Postgres 连接串。
// 不用 URL 形式：URL 会把 TimeZone 等值编码成 %2F，pgx 不会自动解码。
func (d DBConfig) DSN() string {
	kv := []string{
		"host=" + dsnValue(d.Host),
		"port=" + dsnValue(d.Port),
		"user=" + dsnValue(d.User),
		"password=" + dsnValue(d.Password),
		"dbname=" + dsnValue(d.Name),
		"sslmode=" + dsnValue(d.SSLMode),
		// TimeZone 是 pgx 的 runtime 参数，不会去掉引号，只能原样传
		"TimeZone=" + d.TimeZone,
	}
	return strings.Join(kv, " ")
}

// dsnValue 按 libpq conninfo 规则加引号：反斜杠与单引号需要转义。
func dsnValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s=%q 不是合法整数", key, raw)
	}
	return v, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s=%q 不是合法时长（如 30m）", key, raw)
	}
	return v, nil
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

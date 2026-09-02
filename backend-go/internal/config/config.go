package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config 应用配置，全部来自环境变量，默认值适合本地开发。
type Config struct {
	ProjectName string
	Port        string
	SecretKey   string

	DatabasePath string // SQLite 数据库文件路径（新应用）
	UploadDir    string // 上传文件存储目录（新应用）

	LegacyDBPath     string // 旧版 life-recorder 数据库文件路径（迁移用）
	LegacyUploadsDir string // 旧版 life-recorder 上传目录（迁移用）

	AdminEmails string // 管理员邮箱（逗号分隔），启动时自动提升为管理员

	AppBaseURL string // 前端访问地址（用于生成找回密码链接）
	SMTPHost   string // SMTP 服务器地址（留空表示未配置邮件，将进入开发模式）
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string     // 发件人邮箱
	SMTPFromName string   // 发件人显示名
}

// Load 读取 .env 与环境变量，构造配置。
func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		ProjectName:      getEnv("PROJECT_NAME", "飞光 Lumifly"),
		Port:             getEnv("PORT", "8004"),
		SecretKey:        getEnv("SECRET_KEY", "please-change-me-to-a-random-secret"),
		DatabasePath:     getEnv("DATABASE_PATH", "./data/lumifly.db"),
		UploadDir:        getEnv("UPLOAD_DIR", "./uploads"),
		LegacyDBPath:     getEnv("LEGACY_DB_PATH", ""),
		LegacyUploadsDir: getEnv("LEGACY_UPLOADS_DIR", ""),
		AdminEmails:      getEnv("ADMIN_EMAILS", ""),
		AppBaseURL:       getEnv("APP_BASE_URL", "http://localhost:5176"),
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         getEnv("SMTP_PORT", "587"),
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPass:         getEnv("SMTP_PASS", ""),
		SMTPFrom:         getEnv("SMTP_FROM", ""),
		SMTPFromName:     getEnv("SMTP_FROM_NAME", "飞光 Lumifly"),
	}

	if cfg.SecretKey == "please-change-me-to-a-random-secret" {
		log.Println("[warn] 请修改 SECRET_KEY 为强随机值")
	}
	return cfg
}

// ResolvePath 将相对路径解析为相对于后端运行目录的绝对路径。
func ResolvePath(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

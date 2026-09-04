package database

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"lumiflybackend/internal/config"
	"lumiflybackend/internal/model"
)

// Open 打开 SQLite 数据库。
// 首次启动时若目标库不存在且配置了旧版 life-recorder 数据库，则自动复制旧库完成数据迁移。
func Open(cfg *config.Config) (*gorm.DB, error) {
	dbPath := config.ResolvePath(cfg.DatabasePath)

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	// 数据迁移：新库不存在时，从旧版库复制
	if !fileExists(dbPath) {
		if cfg.LegacyDBPath != "" {
			legacy := config.ResolvePath(cfg.LegacyDBPath)
			if fileExists(legacy) {
				if err := copyFile(legacy, dbPath); err != nil {
					return nil, fmt.Errorf("copy legacy db: %w", err)
				}
				log.Printf("[迁移] 已从旧版数据库复制数据：%s → %s", legacy, dbPath)
			} else {
				log.Printf("[迁移] 未找到旧版数据库 %s，跳过（将新建空库）", legacy)
			}
		}
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	// SQLite 单文件：限制连接数避免写锁冲突
	sqlDB.SetMaxOpenConns(1)

	// 兼容旧库行为
	db.Exec("PRAGMA foreign_keys = ON")

	return db, nil
}

// Migrate 建表。
// 关键：若库中已存在旧版 life-recorder 的表结构（首次迁移），必须跳过 AutoMigrate，
// 否则 GORM 会因列类型细微差异（如 TEXT vs VARCHAR）执行 SQLite 表重建而清空数据。
// 仅在全新数据库时才执行自动建表。
func Migrate(db *gorm.DB) error {
	if db.Migrator().HasTable(&model.User{}) {
		log.Println("[迁移] 检测到已有数据库表结构，跳过自动建表以保护既有数据")
		return nil
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Milestone{},
		&model.Record{},
		&model.Plan{},
		&model.Idea{},
		&model.Media{},
		&model.Tag{},
		&model.RecordTag{},
		&model.MilestoneTag{},
		&model.IdeaTag{},
		&model.PlanTag{},
		&model.SystemSetting{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// EnsureSchema 对既有数据库执行轻量 schema 升级：
//   - 禁止使用 AutoMigrate（会因列差异重建表而清空数据），改用原生 ALTER TABLE ADD COLUMN。
//   - 升级 users 表：新增 role / is_active 列。
//   - 确保 system_settings 表存在。
//   - 依据配置把指定邮箱提升为管理员。
func EnsureSchema(db *gorm.DB, cfg *config.Config) error {
	// 1) users 新增 role
	if err := db.Exec("ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'").Error; err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("add role column: %w", err)
		}
	}
	// 2) users 新增 is_active
	if err := db.Exec("ALTER TABLE users ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1").Error; err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("add is_active column: %w", err)
		}
	}
	// 2.1) users 新增密码找回相关列（均可空，无默认值）
	for _, col := range []struct {
		name string
		typ  string
	}{
		{"password_hint", "TEXT"},
		{"security_question", "TEXT"},
		{"security_answer", "TEXT"},
		{"signature", "TEXT"},
	} {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s", col.name, col.typ)).Error; err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				return fmt.Errorf("add %s column: %w", col.name, err)
			}
		}
	}
	// 3) system_settings 表（全新表，用 AutoMigrate 是安全的）
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		return fmt.Errorf("create settings table: %w", err)
	}
	// 3.1) 读书记录 books / book_tags 表（全新表，只迁移这两个模型，不影响旧表）
	if err := db.AutoMigrate(&model.Book{}, &model.BookTag{}); err != nil {
		return fmt.Errorf("create books table: %w", err)
	}
	// 3.2) 书籍领域库 book_domains 表
	if err := db.AutoMigrate(&model.BookDomain{}); err != nil {
		return fmt.Errorf("create book_domains table: %w", err)
	}
	// 3.3) 广场 publications / likes / comments 表
	if err := db.AutoMigrate(&model.Publication{}, &model.PublicationLike{}, &model.PublicationComment{}); err != nil {
		return fmt.Errorf("create publications table: %w", err)
	}
	// 3.4) 邀请注册 invitations 表
	if err := db.AutoMigrate(&model.Invitation{}); err != nil {
		return fmt.Errorf("create invitations table: %w", err)
	}
	// 4) 提升管理员
	if emails := splitCSV(cfg.AdminEmails); len(emails) > 0 {
		res := db.Model(&model.User{}).Where("email IN ?", emails).Update("role", model.RoleAdmin)
		if res.Error != nil {
			return fmt.Errorf("promote admin: %w", res.Error)
		}
		if res.RowsAffected > 0 {
			log.Printf("[迁移] 已将 %d 个账号提升为管理员", res.RowsAffected)
		}
	}
	return nil
}

func splitCSV(s string) []string {
	parts := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// MigrateUploads 首次启动时把旧版上传目录（图片/视频）复制到新目录。
func MigrateUploads(cfg *config.Config) {
	dst := config.ResolvePath(cfg.UploadDir)
	if fileExists(dst) || cfg.LegacyUploadsDir == "" {
		return
	}
	src := config.ResolvePath(cfg.LegacyUploadsDir)
	if !fileExists(src) {
		log.Printf("[迁移] 未找到旧版上传目录 %s，跳过", src)
		return
	}
	if err := copyDir(src, dst); err != nil {
		log.Printf("[迁移] 复制上传目录失败：%v", err)
		return
	}
	log.Printf("[迁移] 已复制旧版上传文件：%s → %s", src, dst)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

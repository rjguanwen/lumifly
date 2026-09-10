package database

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/url"
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

// dsnWithPragmas 为库文件路径附加连接级 PRAGMA。
// 底层驱动（github.com/glebarez/go-sqlite）在**每条新连接**建立时依次执行这些 pragma
// （见其 applyQueryParams），因此即使放开连接池，设置也对每个连接生效。
// 这一点是 db.Exec("PRAGMA ...") 做不到的——后者只作用于它借到的那一条连接。
func dsnWithPragmas(path string) string {
	// 顺序有意为之：先定日志模式，再调与该模式相关的项（synchronous、锁等待）。
	pragmas := []string{
		// WAL：允许「多读 + 单写」并发，读不再阻塞写、写不再阻塞读。
		// 该设置持久化在库头内，这里显式声明以免依赖旧库遗留状态。
		"journal_mode(WAL)",
		// WAL 下的官方推荐档位：不再每次提交都 fsync，写延迟显著下降且不会因掉电损坏库。
		"synchronous(NORMAL)",
		// 与驱动默认值一致：遇到写锁竞争时等待而非立即返回 SQLITE_BUSY。
		"busy_timeout(5000)",
		// 外键约束（原先由 db.Exec 设置，现移到连接级以保证所有连接一致）。
		"foreign_keys(1)",
		// 负值表示 KiB：约 20MB 页缓存，减少同一请求内重复读盘的次数。
		"cache_size(-20000)",
	}
	parts := make([]string, 0, len(pragmas))
	for _, p := range pragmas {
		parts = append(parts, "_pragma="+url.QueryEscape(p))
	}
	// 显式事务以 BEGIN IMMEDIATE 开启：直接拿写锁，避免「先读后写」在升级写锁时
	// 拿到不会等待重试的 SQLITE_BUSY（WAL 下该场景 busy handler 不生效）。
	parts = append(parts, "_txlock=immediate")
	return path + "?" + strings.Join(parts, "&")
}

// execAddColumn 执行幂等 ALTER ADD COLUMN：列已存在（duplicate column）视为成功，避免噪音日志。
func execAddColumn(sqlDB *sql.DB, query string) error {
	if _, err := sqlDB.Exec(query); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	return nil
}

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

	// 注意：DSN 只被驱动的 Open 解析（? 后的部分不会参与文件名），库文件路径本身不受影响。
	db, err := gorm.Open(sqlite.Open(dsnWithPragmas(dbPath)), &gorm.Config{
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
	// 连接池：SQLite 在 WAL 模式下支持并发读，把池限制为 1 会让所有请求的全部 SQL（包括
	// 同一请求内的多条查询）互相排队，成为吞吐的硬上限。这里放开为少量并发连接，
	// 写冲突由上面的 WAL + busy_timeout 处理；_txlock(immediate) 则避免「先读后写」的
	// 事务在升级写锁时直接拿到 SQLITE_BUSY（该场景下 busy handler 不会等待）。
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	// SQLite 连接无服务端超时，保持长连接复用，避免反复打开文件与重跑 pragma。
	sqlDB.SetConnMaxLifetime(0)

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
	// 0) 底层连接：用户表列升级用 database/sql 直连，可静默处理“重复列”等已知迁移错误，
	//    避免 GORM logger 把每次启动的幂等 ALTER 误报为错误日志。
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}
	// 1) users 新增 role
	if err := execAddColumn(sqlDB, "ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'"); err != nil {
		return fmt.Errorf("add role column: %w", err)
	}
	// 2) users 新增 is_active
	if err := execAddColumn(sqlDB, "ALTER TABLE users ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1"); err != nil {
		return fmt.Errorf("add is_active column: %w", err)
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
		{"square_read_at", "TEXT"},
	} {
		if err := execAddColumn(sqlDB, fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s", col.name, col.typ)); err != nil {
			return fmt.Errorf("add %s column: %w", col.name, err)
		}
	}
	// 2.2) 广场已读水位回填（新功能上线前发布的内容不算未读）
	if err := db.Exec("UPDATE users SET square_read_at = ? WHERE square_read_at IS NULL OR square_read_at = ''", model.NowISO()).Error; err != nil {
		return fmt.Errorf("backfill square_read_at: %w", err)
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
	// 3.5) 速记语录 quick_notes / quick_note_tags 表
	if err := db.AutoMigrate(&model.QuickNote{}, &model.QuickNoteTag{}); err != nil {
		return fmt.Errorf("create quick notes table: %w", err)
	}
	// 3.6) 好友 friendships 表
	if err := db.AutoMigrate(&model.Friendship{}); err != nil {
		return fmt.Errorf("create friendships table: %w", err)
	}
	// 3.7) 人生日历分享 calendar_shares / calendar_comments / calendar_comment_reads 表
	if err := db.AutoMigrate(&model.CalendarShare{}, &model.CalendarComment{}, &model.CalendarCommentRead{}); err != nil {
		return fmt.Errorf("create calendar share tables: %w", err)
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

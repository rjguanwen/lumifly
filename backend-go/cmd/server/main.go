package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/config"
	"lumiflybackend/internal/database"
	"lumiflybackend/internal/handler"
	"lumiflybackend/internal/middleware"
)

func main() {
	cfg := config.Load()

	// 旧版上传目录迁移（仅首次）
	database.MigrateUploads(cfg)

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if err := database.EnsureSchema(db, cfg); err != nil {
		log.Fatalf("upgrade schema: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	auth := middleware.NewAuth(cfg.SecretKey, db)
	hh := handler.New(db, cfg, auth)
	hh.RegisterRoutes(r)

	addr := ":" + cfg.Port
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("%s 已启动，监听 %s", cfg.ProjectName, addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("run server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("正在停止服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

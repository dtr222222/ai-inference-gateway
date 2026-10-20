package repository

import (
	"ai-inference-gateway/internal/model"
	"context"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(dsn string, ctx context.Context) (*gorm.DB, error) {
	// docker-compose 已经保证 mysql 一定会优先启动，但等之后脱离 docker-compose 时单用 gorm.Open 不能保证
	// mysql 数据库真的被连上，因为 gorm.open 是懒加载机制，只有在第一次使用时才会真的去链接
	// 因此我们要提前检测一下是不是真的能连接上 （防御性编程）
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	err = sqlDB.PingContext(ctxTimeout)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&model.CheckResult{})
	if err != nil {
		return nil, err
	}

	// 连接的最长空闲的时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	// 连接的最长存活时间
	sqlDB.SetConnMaxLifetime(time.Hour)
	// 限制最大并发连接数
	sqlDB.SetMaxOpenConns(100)
	// 限制最大空闲连接处
	sqlDB.SetMaxIdleConns(100)

	return db, nil
}

package repository

import (
	"my-checker/internal/model"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&model.CheckResult{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
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

package model

import "time"

type CheckResult struct {
	// 这里不太推荐使用 gorm.Model，会增加无用而可能出错的参数
	ID        uint `gorm:"primaryKey"`
	URL       string
	Status    string
	Success   bool
	Duration  int64
	ErrMsg    string
	CreatedAt time.Time
}

package model

import "time"

type CheckResult struct {
	// 这里不太推荐使用 gorm.Model，会增加无用而可能出错的参数
	ID       uint `gorm:"primaryKey"`
	URL      string
	Status   string
	Success  bool
	Duration int64
	ErrMsg   string
	// 加了 gorm:"index“ 后，GORM 会在这个字段上建一棵类似“字典目录”的 B+ 树。
	// 这棵树里的数据是排好序的，想取数据（倒序），
	//数据库直接跑到 B+ 树的尾巴上，顺着链表往前倒着走，就直接拿到了。
	CreatedAt time.Time `gorm:"index"`
}

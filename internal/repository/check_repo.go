package repository

import (
	"context"
	"my-checker/internal/model"

	"gorm.io/gorm"
)

type CheckRepository struct {
	DB *gorm.DB
}

func NewCheckRepository(db *gorm.DB) *CheckRepository {
	return &CheckRepository{db}
}

func (cr *CheckRepository) SaveResults(ctx context.Context, results []model.CheckResult) error {
	db := cr.DB.WithContext(ctx).Create(&results)
	return db.Error
}

func (cr *CheckRepository) GetResults(ctx context.Context, limit, offset int) ([]model.CheckResult, error) {
	var results []model.CheckResult
	db := cr.DB.WithContext(ctx).Limit(limit).Offset(offset).Order("created_at DESC").Find(&results)
	return results, db.Error
}

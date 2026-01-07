package services

import (
	"TapTransit-backend/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CleanupService 数据清理服务（如 TapEvent 过期数据）。
type CleanupService struct {
	db *gorm.DB
}

// NewCleanupService 创建清理服务。
func NewCleanupService(db *gorm.DB) *CleanupService {
	return &CleanupService{
		db: db,
	}
}

// CleanupTapEvents 清理过期的 TapEvent 记录（默认保留 7 天）。
func (s *CleanupService) CleanupTapEvents(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 7 // 默认保留7天
	}

	// 计算截止时间（保留最近 N 天的数据）
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 删除过期记录
	result := s.db.Where("created_at < ?", cutoffTime).Delete(&models.TapEvent{})
	if result.Error != nil {
		return 0, fmt.Errorf("清理TapEvent失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// StartCleanupTask 启动数据清理定时任务。
func (s *CleanupService) StartCleanupTask(intervalHours int, retentionDays int) {
	if intervalHours <= 0 {
		intervalHours = 24 // 默认每24小时执行一次
	}
	if retentionDays <= 0 {
		retentionDays = 7 // 默认保留7天
	}

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	go func() {
		// 周期清理过期数据
		for range ticker.C {
			count, err := s.CleanupTapEvents(retentionDays)
			if err != nil {
				fmt.Printf("清理TapEvent数据失败: %v\n", err)
			} else if count > 0 {
				fmt.Printf("清理了 %d 条过期的TapEvent记录\n", count)
			}
		}
	}()
}

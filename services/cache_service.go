// Package services 聚合业务逻辑与定时任务。
package services

import (
	"TapTransit-backend/models"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// CacheService 配置数据缓存服务（线路/站点/黑名单）。
type CacheService struct {
	db *gorm.DB // 数据库连接

	// 缓存数据
	routesCache     map[uint]*models.Route
	routesCacheTime time.Time
	routesMutex     sync.RWMutex

	stationsCache     map[uint]*models.Station
	stationsCacheTime time.Time
	stationsMutex     sync.RWMutex

	blacklistCache     map[string]bool
	blacklistCacheTime time.Time
	blacklistMutex     sync.RWMutex

	// 缓存有效期（分钟）
	cacheExpiryMinutes int
}

// NewCacheService 创建缓存服务并设置默认缓存过期时间。
func NewCacheService(db *gorm.DB) *CacheService {
	return &CacheService{
		db:                 db,
		routesCache:        make(map[uint]*models.Route),
		stationsCache:      make(map[uint]*models.Station),
		blacklistCache:     make(map[string]bool),
		cacheExpiryMinutes: 10, // 默认10分钟过期
	}
}

// GetRoute 获取线路信息（带缓存，过期自动回源数据库）。
func (s *CacheService) GetRoute(routeID uint) (*models.Route, error) {
	s.routesMutex.RLock()
	route, exists := s.routesCache[routeID]
	cacheTime := s.routesCacheTime
	s.routesMutex.RUnlock()

	// 如果缓存存在且未过期
	if exists && time.Since(cacheTime) < time.Duration(s.cacheExpiryMinutes)*time.Minute {
		return route, nil
	}

	// 从数据库加载
	var routeModel models.Route
	if err := s.db.First(&routeModel, routeID).Error; err != nil {
		return nil, fmt.Errorf("线路不存在: %w", err)
	}

	// 更新缓存（写锁）
	s.routesMutex.Lock()
	s.routesCache[routeID] = &routeModel
	s.routesCacheTime = time.Now()
	s.routesMutex.Unlock()

	return &routeModel, nil
}

// GetStation 获取站点信息（带缓存，过期自动回源数据库）。
func (s *CacheService) GetStation(stationID uint) (*models.Station, error) {
	s.stationsMutex.RLock()
	station, exists := s.stationsCache[stationID]
	cacheTime := s.stationsCacheTime
	s.stationsMutex.RUnlock()

	// 如果缓存存在且未过期
	if exists && time.Since(cacheTime) < time.Duration(s.cacheExpiryMinutes)*time.Minute {
		return station, nil
	}

	// 从数据库加载
	var stationModel models.Station
	if err := s.db.First(&stationModel, stationID).Error; err != nil {
		return nil, fmt.Errorf("站点不存在: %w", err)
	}

	// 更新缓存（写锁）
	s.stationsMutex.Lock()
	s.stationsCache[stationID] = &stationModel
	s.stationsCacheTime = time.Now()
	s.stationsMutex.Unlock()

	return &stationModel, nil
}

// IsBlacklisted 检查卡片是否在黑名单中（带缓存）。
func (s *CacheService) IsBlacklisted(cardID string) (bool, error) {
	s.blacklistMutex.RLock()
	isBlacklisted, exists := s.blacklistCache[cardID]
	cacheTime := s.blacklistCacheTime
	s.blacklistMutex.RUnlock()

	// 如果缓存存在且未过期
	if exists && time.Since(cacheTime) < time.Duration(s.cacheExpiryMinutes)*time.Minute {
		return isBlacklisted, nil
	}

	// 从数据库加载（查询 cards 表中 status 为 blocked/lost 的卡片）
	var card models.Card
	err := s.db.Where("card_id = ? AND (status = 'blocked' OR status = 'lost')", cardID).First(&card).Error
	isBlacklisted = (err == nil) // 如果找到记录，说明在黑名单中

	// 更新缓存
	s.blacklistMutex.Lock()
	s.blacklistCache[cardID] = isBlacklisted
	s.blacklistCacheTime = time.Now()
	s.blacklistMutex.Unlock()

	return isBlacklisted, nil
}

// RefreshRoutesCache 刷新线路缓存（全量）。
func (s *CacheService) RefreshRoutesCache() error {
	var routes []models.Route
	if err := s.db.Where("status = ?", "active").Find(&routes).Error; err != nil {
		return fmt.Errorf("刷新线路缓存失败: %w", err)
	}

	s.routesMutex.Lock()
	s.routesCache = make(map[uint]*models.Route)
	for i := range routes {
		s.routesCache[routes[i].ID] = &routes[i]
	}
	s.routesCacheTime = time.Now()
	s.routesMutex.Unlock()

	return nil
}

// RefreshStationsCache 刷新站点缓存（全量）。
func (s *CacheService) RefreshStationsCache() error {
	var stations []models.Station
	if err := s.db.Find(&stations).Error; err != nil {
		return fmt.Errorf("刷新站点缓存失败: %w", err)
	}

	s.stationsMutex.Lock()
	s.stationsCache = make(map[uint]*models.Station)
	for i := range stations {
		s.stationsCache[stations[i].ID] = &stations[i]
	}
	s.stationsCacheTime = time.Now()
	s.stationsMutex.Unlock()

	return nil
}

// RefreshBlacklistCache 刷新黑名单缓存（全量）。
func (s *CacheService) RefreshBlacklistCache() error {
	var blockedCards []models.Card
	if err := s.db.Where("status = ? OR status = ?", "blocked", "lost").Find(&blockedCards).Error; err != nil {
		return fmt.Errorf("刷新黑名单缓存失败: %w", err)
	}

	s.blacklistMutex.Lock()
	s.blacklistCache = make(map[string]bool)
	for _, card := range blockedCards {
		s.blacklistCache[card.CardID] = true
	}
	s.blacklistCacheTime = time.Now()
	s.blacklistMutex.Unlock()

	return nil
}

// StartCacheRefreshTask 启动缓存刷新定时任务（定时全量刷新）。
func (s *CacheService) StartCacheRefreshTask(intervalMinutes int) {
	if intervalMinutes <= 0 {
		intervalMinutes = 5 // 默认每5分钟刷新一次
	}

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	go func() {
		// 启动时立即刷新一次
		s.RefreshRoutesCache()
		s.RefreshStationsCache()
		s.RefreshBlacklistCache()

		// 周期刷新
		for range ticker.C {
			s.RefreshRoutesCache()
			s.RefreshStationsCache()
			s.RefreshBlacklistCache()
		}
	}()
}

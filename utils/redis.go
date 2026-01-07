package utils

import (
	"context"
	"TapTransit-backend/config"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient 为全局 Redis 连接句柄。
var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接并做 Ping 校验。
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试连接
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	RedisClient = rdb
	return rdb, nil
}

// GetCardMonthlyAmount 获取卡片当月累计金额（不存在则返回 0）。
func GetCardMonthlyAmount(cardID string) (float64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("card:monthly:%s:%s", time.Now().Format("2006-01"), cardID)
	val, err := RedisClient.Get(ctx, key).Float64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// SetCardMonthlyAmount 设置卡片当月累计金额，过期时间略大于一个月。
func SetCardMonthlyAmount(cardID string, amount float64) error {
	ctx := context.Background()
	key := fmt.Sprintf("card:monthly:%s:%s", time.Now().Format("2006-01"), cardID)
	return RedisClient.Set(ctx, key, amount, 32*24*time.Hour).Err() // 保存32天
}

// IncrementCardMonthlyAmount 以增量方式更新当月累计金额。
func IncrementCardMonthlyAmount(cardID string, amount float64) error {
	ctx := context.Background()
	key := fmt.Sprintf("card:monthly:%s:%s", time.Now().Format("2006-01"), cardID)
	return RedisClient.IncrByFloat(ctx, key, amount).Err()
}

// GetCardOnboardInfo 获取卡片最近一次上车信息（用于换乘判断）。
func GetCardOnboardInfo(cardID string) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("card:onboard:%s", cardID)
	return RedisClient.Get(ctx, key).Result()
}

// SetCardOnboardInfo 设置卡片上车信息（用于换乘判断）。
func SetCardOnboardInfo(cardID string, info string, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("card:onboard:%s", cardID)
	return RedisClient.Set(ctx, key, info, ttl).Err()
}

// DeleteCardOnboardInfo 删除卡片上车信息。
func DeleteCardOnboardInfo(cardID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("card:onboard:%s", cardID)
	return RedisClient.Del(ctx, key).Err()
}

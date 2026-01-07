// Package config 负责读取并解析后端运行所需的配置（服务、数据库、缓存、日志）。
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config 是应用级配置的聚合结构体，直接映射 config.yaml。
type Config struct {
	Server   ServerConfig   `yaml:"server"`   // HTTP 服务监听配置
	Database DatabaseConfig `yaml:"database"` // PostgreSQL 连接配置
	Redis    RedisConfig    `yaml:"redis"`    // Redis 连接配置
	Logging  LoggingConfig  `yaml:"logging"`  // 日志输出配置
}

// ServerConfig 描述 Gin 服务的基础配置。
type ServerConfig struct {
	Host string `yaml:"host"` // 监听地址
	Port int    `yaml:"port"` // 监听端口
	Mode string `yaml:"mode"` // Gin 运行模式（debug/release）
}

// DatabaseConfig 描述数据库连接与连接池配置。
type DatabaseConfig struct {
	Host            string `yaml:"host"`              // 数据库主机
	Port            int    `yaml:"port"`              // 数据库端口
	User            string `yaml:"user"`              // 登录用户名
	Password        string `yaml:"password"`          // 登录密码
	DBName          string `yaml:"dbname"`            // 数据库名称
	SSLMode         string `yaml:"sslmode"`           // SSL 模式（disable/require）
	Timezone        string `yaml:"timezone"`          // 会话时区
	MaxOpenConns    int    `yaml:"max_open_conns"`    // 最大连接数
	MaxIdleConns    int    `yaml:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // 连接最大存活时间（秒）
}

// RedisConfig 描述 Redis 连接与连接池参数。
type RedisConfig struct {
	Host     string `yaml:"host"`      // Redis 主机
	Port     int    `yaml:"port"`      // Redis 端口
	Password string `yaml:"password"`  // Redis 密码（可为空）
	DB       int    `yaml:"db"`        // 使用的逻辑库编号
	PoolSize int    `yaml:"pool_size"` // 连接池大小
}

// LoggingConfig 描述日志输出级别与格式。
type LoggingConfig struct {
	Level  string `yaml:"level"`  // 日志级别（info/debug/error）
	Format string `yaml:"format"` // 输出格式（json/text）
}

// AppConfig 保存全局配置，供其他包读取。
var AppConfig *Config

// LoadConfig 加载并解析 YAML 配置文件。
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	AppConfig = config
	return config, nil
}

// GetDSN 拼接 PostgreSQL DSN 字符串。
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Password, c.DBName, c.Port, c.SSLMode, c.Timezone)
}

// GetRedisAddr 生成 Redis 的 host:port 地址。
func (r *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// InitDB 初始化数据库连接与连接池，并进行连通性校验。
func InitDB(config *DatabaseConfig) (*gorm.DB, error) {
	dsn := config.GetDSN()

	// 配置 GORM 选项
	gormConfig := &gorm.Config{
		// 禁用外键约束检查（迁移时），让 GORM 自动处理外键创建顺序
		DisableForeignKeyConstraintWhenMigrating: false, // 保持为false，让GORM处理
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Second)
	}

	// 测试数据库连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	return db, nil
}

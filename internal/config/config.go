package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	R2       R2Config
	JWT      JWTConfig
	SMS      SMSConfig
	WeChat   WeChatConfig
	GLM      GLMConfig
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Mode         string        `mapstructure:"mode"` // debug, release, test
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Enabled  bool   `mapstructure:"enabled"`
}

type R2Config struct {
	AccountID       string `mapstructure:"account_id"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Bucket          string `mapstructure:"bucket"`
	PublicURL       string `mapstructure:"public_url"`
}

type JWTConfig struct {
	Secret            string        `mapstructure:"secret"`
	AccessExpiration  time.Duration `mapstructure:"access_expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration"`
}

type SMSConfig struct {
	Provider  string `mapstructure:"provider"`
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	SignName  string `mapstructure:"sign_name"`
}

type WeChatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

type GLMConfig struct {
	APIKey  string        `mapstructure:"api_key"`
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Read .env file (optional — env vars take precedence)
	_ = viper.ReadInConfig()

	viper.SetEnvPrefix("SPATIAL")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetInt("server.port"),
			ReadTimeout:  viper.GetDuration("server.read_timeout"),
			WriteTimeout: viper.GetDuration("server.write_timeout"),
			Mode:         viper.GetString("server.mode"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("database.host"),
			Port:            viper.GetInt("database.port"),
			User:            viper.GetString("database.user"),
			Password:        viper.GetString("database.password"),
			DBName:          viper.GetString("database.dbname"),
			SSLMode:         viper.GetString("database.sslmode"),
			MinConns:        int32(viper.GetInt("database.min_conns")),
			MaxConns:        int32(viper.GetInt("database.max_conns")),
			MaxConnLifetime: viper.GetDuration("database.max_conn_lifetime"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetInt("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
			Enabled:  viper.GetBool("redis.enabled"),
		},
		R2: R2Config{
			AccountID:       viper.GetString("r2.account_id"),
			AccessKeyID:     viper.GetString("r2.access_key_id"),
			AccessKeySecret: viper.GetString("r2.access_key_secret"),
			Bucket:          viper.GetString("r2.bucket"),
			PublicURL:       viper.GetString("r2.public_url"),
		},
		JWT: JWTConfig{
			Secret:            viper.GetString("jwt.secret"),
			AccessExpiration:  viper.GetDuration("jwt.access_expiration"),
			RefreshExpiration: viper.GetDuration("jwt.refresh_expiration"),
		},
		SMS: SMSConfig{
			Provider:  viper.GetString("sms.provider"),
			APIKey:    viper.GetString("sms.api_key"),
			APISecret: viper.GetString("sms.api_secret"),
			SignName:  viper.GetString("sms.sign_name"),
		},
		WeChat: WeChatConfig{
			AppID:     viper.GetString("wechat.app_id"),
			AppSecret: viper.GetString("wechat.app_secret"),
		},
		GLM: GLMConfig{
			APIKey:  viper.GetString("glm.api_key"),
			BaseURL: viper.GetString("glm.base_url"),
			Timeout: viper.GetDuration("glm.timeout"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 10*time.Second)
	viper.SetDefault("server.write_timeout", 10*time.Second)
	viper.SetDefault("server.mode", "debug")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "spatial")
	viper.SetDefault("database.password", "spatial")
	viper.SetDefault("database.dbname", "spatial_memory")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.min_conns", 2)
	viper.SetDefault("database.max_conns", 20)
	viper.SetDefault("database.max_conn_lifetime", 30*time.Minute)

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.enabled", true)

	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.access_expiration", 2*time.Hour)
	viper.SetDefault("jwt.refresh_expiration", 30*24*time.Hour)

	viper.SetDefault("glm.base_url", "https://open.bigmodel.cn/api/paas/v4")
	viper.SetDefault("glm.timeout", 30*time.Second)
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

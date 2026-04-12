package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"app/pkg/i18n"
	"app/pkg/logger"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Log        LogConfig        `mapstructure:"log"`
	Cache      CacheConfig      `mapstructure:"cache"`
	Queue      QueueConfig      `mapstructure:"queue"`
	I18n       i18n.Config      `mapstructure:"i18n"`
	CORS       CORSConfig       `mapstructure:"cors"`
	Server     ServerConfig     `mapstructure:"server"`
	Storage    StorageConfig    `mapstructure:"storage"`
	SuperAdmin SuperAdminConfig `mapstructure:"super_admin"`
	// SuperAdminIDs parsed once at load (from super_admin.user_ids). Not loaded from YAML keys.
	SuperAdminIDs []uint `yaml:"-" mapstructure:"-"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Address string `mapstructure:"address"`
	Mode    string `mapstructure:"mode"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name      string `mapstructure:"name"`
	Mode      string `mapstructure:"mode"`
	Port      int    `mapstructure:"port"`
	APIPrefix string `mapstructure:"api_prefix"`
	Env       string `mapstructure:"env"`
	Debug     bool   `mapstructure:"debug"`
	BaseURL   string `mapstructure:"baseUrl"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireTime int    `mapstructure:"expire_time"`
	Issuer     string `mapstructure:"issuer"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Charset         string `mapstructure:"charset"`
	SSLMode         string `mapstructure:"sslmode"`  // postgres DSN sslmode (e.g. disable, require)
	TimeZone        string `mapstructure:"timezone"` // postgres DSN TimeZone (e.g. UTC, Asia/Shanghai)
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`             // 日志级别
	Filename   string `yaml:"filename" mapstructure:"filename"`       // 日志文件路径
	MaxSize    int    `yaml:"max_size" mapstructure:"max_size"`       // 每个日志文件最大尺寸，单位MB
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`         // 保留的旧日志文件最大天数
	Compress   bool   `yaml:"compress" mapstructure:"compress"`       // 是否压缩旧日志文件
	Daily      bool   `yaml:"daily" mapstructure:"daily"`             // 是否按天切割日志
	Timezone   string `yaml:"timezone" mapstructure:"timezone"`       // IANA 时区，按日轮转与 max_age 清理；空或 Local = 进程默认
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Driver  string                 `mapstructure:"driver"`
	Prefix  string                 `mapstructure:"prefix"`
	Options map[string]interface{} `mapstructure:"options"`
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	Driver     string `mapstructure:"driver"`
	Queue      string `mapstructure:"queue"`
	Connection struct {
		Redis    string `mapstructure:"redis"`
		Database string `mapstructure:"database"`
	} `mapstructure:"connection"`
	Worker struct {
		Sleep   int `mapstructure:"sleep"`
		MaxJobs int `mapstructure:"max_jobs"`
		MaxTime int `mapstructure:"max_time"`
		Rest    int `mapstructure:"rest"`
		Memory  int `mapstructure:"memory"`
		Tries   int `mapstructure:"tries"`
		Timeout int `mapstructure:"timeout"`
	} `mapstructure:"worker"`
	Queues map[string]QueueDetail `mapstructure:"queues"`
}

// QueueDetail holds configuration for individual queues
type QueueDetail struct {
	Priority   int   `mapstructure:"priority"`
	Processes  int   `mapstructure:"processes"`
	Timeout    int   `mapstructure:"timeout"`
	Tries      int   `mapstructure:"tries"`
	RetryAfter int   `mapstructure:"retry_after"`
	Backoff    []int `mapstructure:"backoff"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string      `mapstructure:"allow_origins"`
	AllowMethods     []string      `mapstructure:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers"`
	ExposeHeaders    []string      `mapstructure:"expose_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Driver string      `mapstructure:"driver"`
	Local  LocalConfig `mapstructure:"local"`
	S3     S3Config    `mapstructure:"s3"`
}

// LocalConfig holds local storage configuration
type LocalConfig struct {
	Path string `mapstructure:"path"`
}

// S3Config holds S3 storage configuration
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	UseSSL          bool   `mapstructure:"use_ssl"`
}

// SuperAdminConfig holds SuperAdmin configuration
type SuperAdminConfig struct {
	UserIDs []string `mapstructure:"user_ids"`
}

// LoadConfig loads configuration from default search paths and environment variables.
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/app/")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %v", err)
		}
		return nil, fmt.Errorf("error reading config file: %v", err)
	}
	return populateConfigFromViper(viper.GetViper())
}

// LoadConfigFromFile loads configuration from an explicit YAML file (e.g. queue CLI with -config).
func LoadConfigFromFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	return populateConfigFromViper(v)
}

func populateConfigFromViper(v *viper.Viper) (*Config, error) {
	config := &Config{}

	config.App.Name = getEnvOrDefault("APP_NAME", v.GetString("app.name"))
	config.App.Env = getEnvOrDefault("APP_ENV", v.GetString("app.env"))
	config.App.Mode = getEnvOrDefault("APP_MODE", v.GetString("app.mode"))
	config.App.Debug = getEnvBoolOrDefault("APP_DEBUG", v.GetBool("app.debug"))
	config.App.BaseURL = getEnvOrDefault("APP_URL", v.GetString("app.baseUrl"))
	config.App.Port = getEnvIntOrDefault("APP_PORT", v.GetInt("app.port"))
	config.App.APIPrefix = getEnvOrDefault("APP_API_PREFIX", v.GetString("app.api_prefix"))

	config.JWT.Secret = getEnvOrDefault("JWT_SECRET", v.GetString("jwt.secret"))
	config.JWT.ExpireTime = getEnvIntOrDefault("JWT_EXPIRE", v.GetInt("jwt.expire_time"))
	config.JWT.Issuer = getEnvOrDefault("JWT_ISSUER", v.GetString("jwt.issuer"))

	config.Database.Driver = getEnvOrDefault("DB_DRIVER", v.GetString("database.driver"))
	config.Database.Host = getEnvOrDefault("DB_HOST", v.GetString("database.host"))
	config.Database.Port = getEnvOrDefault("DB_PORT", v.GetString("database.port"))
	config.Database.Username = getEnvOrDefault("DB_USERNAME", v.GetString("database.username"))
	config.Database.Password = getEnvOrDefault("DB_PASSWORD", v.GetString("database.password"))
	config.Database.Database = getEnvOrDefault("DB_DATABASE", v.GetString("database.database"))
	config.Database.Charset = getEnvOrDefault("DB_CHARSET", v.GetString("database.charset"))
	config.Database.SSLMode = getEnvOrDefault("DB_SSLMODE", v.GetString("database.sslmode"))
	config.Database.TimeZone = getEnvOrDefault("DB_TIMEZONE", v.GetString("database.timezone"))
	config.Database.MaxIdleConns = getEnvIntOrDefault("DB_MAX_IDLE_CONNS", v.GetInt("database.max_idle_conns"))
	config.Database.MaxOpenConns = getEnvIntOrDefault("DB_MAX_OPEN_CONNS", v.GetInt("database.max_open_conns"))
	config.Database.ConnMaxLifetime = getEnvIntOrDefault("DB_CONN_MAX_LIFETIME", v.GetInt("database.conn_max_lifetime"))

	config.Redis.Host = getEnvOrDefault("REDIS_HOST", v.GetString("redis.host"))
	config.Redis.Port = getEnvOrDefault("REDIS_PORT", v.GetString("redis.port"))
	config.Redis.Password = getEnvOrDefault("REDIS_PASSWORD", v.GetString("redis.password"))
	config.Redis.DB = getEnvIntOrDefault("REDIS_DB", v.GetInt("redis.db"))

	config.Cache.Driver = getEnvOrDefault("CACHE_DRIVER", v.GetString("cache.driver"))
	config.Cache.Prefix = getEnvOrDefault("CACHE_PREFIX", v.GetString("cache.prefix"))
	config.Cache.Options = v.GetStringMap("cache.options")

	config.Queue.Driver = getEnvOrDefault("QUEUE_DRIVER", v.GetString("queue.driver"))
	config.Queue.Queue = getEnvOrDefault("QUEUE_NAME", v.GetString("queue.queue"))
	config.Queue.Connection.Redis = v.GetString("queue.connection.redis")
	config.Queue.Connection.Database = v.GetString("queue.connection.database")
	config.Queue.Worker.Sleep = v.GetInt("queue.worker.sleep")
	config.Queue.Worker.MaxJobs = v.GetInt("queue.worker.max_jobs")
	config.Queue.Worker.MaxTime = v.GetInt("queue.worker.max_time")
	config.Queue.Worker.Rest = v.GetInt("queue.worker.rest")
	config.Queue.Worker.Memory = v.GetInt("queue.worker.memory")
	config.Queue.Worker.Tries = v.GetInt("queue.worker.tries")
	config.Queue.Worker.Timeout = v.GetInt("queue.worker.timeout")

	config.Queue.Queues = make(map[string]QueueDetail)
	for name := range v.GetStringMap("queue.queues") {
		var detail QueueDetail
		if err := v.UnmarshalKey("queue.queues."+name, &detail); err != nil {
			return nil, fmt.Errorf("error unmarshaling queue %s: %v", name, err)
		}
		config.Queue.Queues[name] = detail
	}

	config.Server.Address = getEnvOrDefault("SERVER_ADDRESS", v.GetString("server.address"))
	config.Server.Mode = getEnvOrDefault("SERVER_MODE", v.GetString("server.mode"))

	config.Log.Level = getEnvOrDefault("LOG_LEVEL", v.GetString("log.level"))
	config.Log.Filename = getEnvOrDefault("LOG_FILENAME", v.GetString("log.filename"))
	config.Log.MaxSize = getEnvIntOrDefault("LOG_MAX_SIZE", v.GetInt("log.max_size"))
	config.Log.MaxBackups = getEnvIntOrDefault("LOG_MAX_BACKUPS", v.GetInt("log.max_backups"))
	config.Log.MaxAge = getEnvIntOrDefault("LOG_MAX_AGE", v.GetInt("log.max_age"))
	config.Log.Compress = getEnvBoolOrDefault("LOG_COMPRESS", v.GetBool("log.compress"))
	config.Log.Daily = getEnvBoolOrDefault("LOG_DAILY", v.GetBool("log.daily"))
	config.Log.Timezone = getEnvOrDefault("LOG_TIMEZONE", v.GetString("log.timezone"))

	config.CORS.AllowOrigins = v.GetStringSlice("cors.allow_origins")
	config.CORS.AllowMethods = v.GetStringSlice("cors.allow_methods")
	config.CORS.AllowHeaders = v.GetStringSlice("cors.allow_headers")
	config.CORS.ExposeHeaders = v.GetStringSlice("cors.expose_headers")
	config.CORS.AllowCredentials = v.GetBool("cors.allow_credentials")
	config.CORS.MaxAge = v.GetDuration("cors.max_age")

	config.I18n.DefaultLocale = getEnvOrDefault("I18N_DEFAULT_LOCALE", v.GetString("i18n.default_locale"))
	config.I18n.LoadPath = getEnvOrDefault("I18N_LOAD_PATH", v.GetString("i18n.load_path"))
	config.I18n.AvailableLocales = v.GetStringSlice("i18n.available_locales")

	config.Storage.Driver = getEnvOrDefault("STORAGE_DRIVER", v.GetString("storage.driver"))
	config.Storage.Local.Path = getEnvOrDefault("STORAGE_LOCAL_PATH", v.GetString("storage.local.path"))
	config.Storage.S3.Endpoint = getEnvOrDefault("STORAGE_S3_ENDPOINT", v.GetString("storage.s3.endpoint"))
	config.Storage.S3.AccessKeyID = getEnvOrDefault("STORAGE_S3_ACCESS_KEY_ID", v.GetString("storage.s3.access_key_id"))
	config.Storage.S3.SecretAccessKey = getEnvOrDefault("STORAGE_S3_SECRET_ACCESS_KEY", v.GetString("storage.s3.secret_access_key"))
	config.Storage.S3.Bucket = getEnvOrDefault("STORAGE_S3_BUCKET", v.GetString("storage.s3.bucket"))
	config.Storage.S3.Region = getEnvOrDefault("STORAGE_S3_REGION", v.GetString("storage.s3.region"))
	config.Storage.S3.UseSSL = getEnvBoolOrDefault("STORAGE_S3_USE_SSL", v.GetBool("storage.s3.use_ssl"))

	for _, idStr := range v.GetStringSlice("super_admin.user_ids") {
		config.SuperAdmin.UserIDs = append(config.SuperAdmin.UserIDs, idStr)
	}
	config.SuperAdminIDs = parseSuperAdminUints(config.SuperAdmin.UserIDs)

	return config, nil
}

func parseSuperAdminUints(strs []string) []uint {
	out := make([]uint, 0, len(strs))
	for _, idStr := range strs {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			out = append(out, uint(id))
		} else {
			logger.Sugared().Warnw("invalid super_admin user_id", "id", idStr, "error", err)
		}
	}
	return out
}

// Validate checks production-safety constraints. Call after LoadConfig / LoadConfigFromFile.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.EqualFold(c.App.Env, "production") {
		if len(c.JWT.Secret) < 32 {
			return fmt.Errorf("production: jwt.secret must be at least 32 characters")
		}
		if strings.TrimSpace(c.Database.Host) == "" {
			return fmt.Errorf("production: database.host is required")
		}
		if strings.TrimSpace(c.Database.Database) == "" {
			return fmt.Errorf("production: database.database is required")
		}
		if strings.TrimSpace(c.Redis.Host) == "" {
			return fmt.Errorf("production: redis.host is required")
		}
		for _, o := range c.CORS.AllowOrigins {
			if strings.TrimSpace(o) == "*" {
				return fmt.Errorf("production: cors.allow_origins must not use wildcard \"*\"")
			}
		}
	}
	return nil
}

// SuperAdminUintIDs returns super-admin user IDs parsed at load time.
func (c *Config) SuperAdminUintIDs() []uint {
	if c == nil {
		return nil
	}
	return c.SuperAdminIDs
}

// getEnvOrDefault gets environment variable value or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault gets environment variable as int or returns default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBoolOrDefault gets environment variable as bool or returns default value
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}


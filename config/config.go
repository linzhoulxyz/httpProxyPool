package config

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	redisClient *redis.Client
	db          *gorm.DB
)

type Config struct {
	Name                    string
	DB                      *gorm.DB
	RedisClient             *redis.Client
	onConfigChangeCallbacks []func(e fsnotify.Event)
}

// 初始化配置
func Init(cfgPath string) (*Config, error) {
	c := Config{
		Name: cfgPath,
	}

	// 初始化配置
	if err := c.initConfig(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Config) initConfig() error {
	if c.Name != "" {
		viper.SetConfigFile(c.Name)
	} else {
		// 没有传入配置地址的话使用默认路径
		viper.AddConfigPath("config")
		viper.SetConfigName("application")
	}

	viper.SetConfigType("yaml") // 配置文件格式为yaml
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

// 初始化日志配置
func (c *Config) InitLog() {
	ljack := &lumberjack.Logger{
		Filename:   viper.GetString("log.logger_file"),
		MaxSize:    viper.GetInt("log.log_rotate_size"), // megabytes
		MaxBackups: viper.GetInt("log.log_backup_count"),
		MaxAge:     viper.GetInt("log.log_rotate_date"), //days
		Compress:   false,                               // disabled by default
	}
	mWriter := io.MultiWriter(os.Stdout, ljack)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(mWriter)
	logrus.SetLevel(logrus.DebugLevel)
}

// 初始化Redis
func (c *Config) InitRedis() error {
	var (
		err    error
		client *redis.Client
	)

	host := viper.GetString("redis.host")
	port := viper.GetInt("redis.port")
	auth := viper.GetString("redis.auth")
	db := viper.GetInt("redis.db")

	if len(host) == 0 {
		return errors.New("not set redis config")
	}

	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%v:%v", host, port),
		Password: auth,
		DB:       db,
	})

	_, err = client.Ping().Result()
	if err != nil {
		return err
	}

	// 指定log输出
	if logPath := viper.GetString("redis.log"); logPath != "" {
		if sqlLog, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755); err != nil {
			return err
		} else {
			writers := []io.Writer{
				sqlLog,
				os.Stdout,
			}
			fileAndStdoutWriter := io.MultiWriter(writers...)
			redis.SetLogger(log.New(fileAndStdoutWriter, "\r\n", log.LstdFlags))
		}
	}

	c.RedisClient = client
	redisClient = client

	return nil
}

func (c *Config) InitDB() error {

	var (
		err error
		tx  *gorm.DB
	)
	if adapter := viper.GetString("db.adapter"); adapter == "mysql" {

		tx, err = gorm.Open(adapter, fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=True&loc=Local", viper.GetString("db.user"), viper.GetString("db.password"), viper.GetString("db.host"), viper.GetInt("db.port"), viper.GetString("db.name")))

	} else {
		return errors.New("not supported database adapter")
	}

	if err != nil {
		return err
	}

	// 全局禁用负数表名
	tx.SingularTable(true)

	// 指定数据库的log输出
	if logPath := viper.GetString("db.log"); logPath != "" {
		ljack := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    viper.GetInt("log.log_rotate_size"), // megabytes
			MaxBackups: viper.GetInt("log.log_backup_count"),
			MaxAge:     viper.GetInt("log.log_rotate_date"), //days
			Compress:   false,                               // disabled by default
		}
		mWriter := io.MultiWriter(os.Stdout, ljack)
		tx.SetLogger(log.New(mWriter, "\r\n", log.LstdFlags))
	}

	if viper.GetBool("db.debug") {
		tx.LogMode(true)
	}

	if prefix := viper.GetString("db.table_prefix"); prefix != "" {
		gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
			if strings.Index(defaultTableName, "oc_") == 0 {
				return defaultTableName
			}
			return prefix + defaultTableName
		}
	}

	c.DB = tx
	db = tx

	return nil

}

func (c *Config) AddConfigWatch(fn func(e fsnotify.Event)) {
	c.onConfigChangeCallbacks = append(c.onConfigChangeCallbacks, fn)
}

// 热更新配置文件
func (c *Config) WatchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logrus.Infof("Config file changed: %s", e.Name)
		if err := viper.ReadInConfig(); err != nil {
			logrus.Error("watchConfig ReadInConfig", err)
		} else {
			for _, fn := range c.onConfigChangeCallbacks {
				fn(e)
			}
		}
	})
}

func GetRedis() (*redis.Client, error) {
	if redisClient == nil {
		return nil, errors.New("redis obj is not init")
	}

	return redisClient, nil
}

func GetDb() (*gorm.DB, error) {
	if db == nil {
		return nil, errors.New("db obj is not init")
	}

	return db, nil
}

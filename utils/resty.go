package utils

import (
	"io"
	"os"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *logrus.Logger
var once sync.Once

type CrawlerInfo struct {
	Site      string
	SportType int
	Lang      int
	UrlType   int
}

func getLogger() *logrus.Logger {
	once.Do(func() {
		ljack := &lumberjack.Logger{
			Filename:   viper.GetString("log.resty_file"),
			MaxSize:    viper.GetInt("log.log_rotate_size"), // megabytes
			MaxBackups: viper.GetInt("log.log_backup_count"),
			MaxAge:     viper.GetInt("log.log_rotate_date"), //days
			Compress:   false,                               // disabled by default
		}
		mWriter := io.MultiWriter(os.Stdout, ljack)
		log = logrus.New()
		log.SetFormatter(&logrus.JSONFormatter{})
		log.SetOutput(mWriter)
		log.SetLevel(logrus.DebugLevel)
	})

	return log
}

func NewResty() *resty.Client {
	client := resty.New().
		SetLogger(getLogger()).
		SetDebug(viper.GetBool("crawler.debug"))
	return client
}

func NewRestyRequestChrome(extraHeader map[string]string) *resty.Request {
	header := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.89 Safari/537.36",
	}
	if extraHeader != nil {
		for key, value := range extraHeader {
			header[key] = value
		}
	}

	request := NewResty().R().
		SetHeaders(header)

	return request
}

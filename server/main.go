package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/samuncle-jqk/httpProxyPool/provider/source"

	"github.com/samuncle-jqk/httpProxyPool/provider"

	"github.com/go-redis/redis/v7"

	"github.com/samuncle-jqk/httpProxyPool/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	var (
		cfg *config.Config
		err error
	)

	// 加载配置文件
	if cfg, err = config.Init(""); err != nil {
		panic(err)
	}

	// 初始化日志配置
	cfg.InitLog()

	// 初始化数据库
	if dbErr := cfg.InitDB(); dbErr != nil {
		panic(dbErr)
	}

	// 初始化Redis
	if err := cfg.InitRedis(); err != nil {
		panic(err)
	}

	// 启动清理协程
	go processUpdateProxyPoolCache()
	// 开始更新代理池
	updateHttpProxyPool()

	select {}

	logrus.Info("HTTP PROXY POOL UPDATE SERVER DOWN!")
}

// updateHttpProxyPool 更新HTTP代理池数据
func updateHttpProxyPool() {
	logrus.Info("Begin update http proxy pool data.")

	sourceUse := viper.GetStringSlice("proxy.source_use")
	if len(sourceUse) == 0 {
		logrus.Info("proxy source use empty")
		return
	}

	for _, sourceName := range sourceUse {
		go func(sourceName string) {
			err := startHttpProxyProvider(sourceName)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err":    err,
					"source": sourceName,
				}).Error("startHttpProxyProvider fail")
			}
		}(sourceName)
	}

	return
}

func startHttpProxyProvider(sourceName string) error {
	db, _ := config.GetDb()
	rClient, _ := config.GetRedis()

	switch sourceName {
	case provider.SOURCE_ZDY:
		p := provider.NewStrategy(db, rClient, source.Zdy{})
		p.UpdateHttpProxy()
		return nil
	default:
		return fmt.Errorf("unknown http proxy sourceName: %v", sourceName)
	}
}

// 定时清理已过期的缓存代理
func processUpdateProxyPoolCache() {
	redisClient, err := config.GetRedis()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("processUpdateProxyPoolCache get redis fail.")
		return
	}

	// 清理离当前时间多长分钟内的代理
	cleanInterval := viper.GetInt("proxy.cache_clean_interval") // 秒
	minAfterSecond := viper.GetInt("proxy.cache_clean_second")  // 秒

	cleanRedisExpiredProxy(redisClient, minAfterSecond)
	ticker := time.NewTicker(time.Second * time.Duration(cleanInterval))
	for range ticker.C {
		go cleanRedisExpiredProxy(redisClient, minAfterSecond)
	}
}

func cleanRedisExpiredProxy(client *redis.Client, minAfterSecond int) {
	deleteBefore := time.Now().Unix() + int64(minAfterSecond)
	ret, err := client.ZRemRangeByScore(provider.CACHE_HTTP_PROXY_POOL_KEY, "0", strconv.Itoa(int(deleteBefore))).Result()
	logrus.WithFields(logrus.Fields{
		"ret": ret,
		"err": err,
	}).Info("clean expire cache proxy.")
}

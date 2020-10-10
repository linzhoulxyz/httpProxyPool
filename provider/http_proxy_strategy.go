package provider

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
	"github.com/samuncle-jqk/httpProxyPool/model"
	"github.com/sirupsen/logrus"
)

type HttpProxyStrategy struct {
	context  *HttpProxyStrategyContext
	Provider HttpProxyProvider
}

type HttpProxyStrategyContext struct {
	Db          *gorm.DB
	RedisClient *redis.Client
}

func NewStrategy(db *gorm.DB, rClient *redis.Client, provider HttpProxyProvider) *HttpProxyStrategy {
	return &HttpProxyStrategy{
		context: &HttpProxyStrategyContext{
			Db:          db,
			RedisClient: rClient,
		},
		Provider: provider,
	}
}

func (h HttpProxyStrategy) UpdateHttpProxy() {
	h.Provider.BindWhiteIp()
	time.Sleep(time.Second)

	intervalTime := h.Provider.GetRequestInterval()
	getIpTicker := time.NewTicker(time.Second * time.Duration(intervalTime))
	bindWhiteIpTicker := time.NewTicker(time.Minute * 1)
	for {
		select {
		case <-getIpTicker.C:
			resp := h.Provider.GetProxyIp()
			err := h.SaveIp(resp)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err":    err,
					"source": h.Provider.GetSource(),
				}).Error("request and save http proxy fail")
			}
		case <-bindWhiteIpTicker.C:
			h.Provider.BindWhiteIp()
		}
	}
}

func (h HttpProxyStrategy) SaveIp(resp ProxyIpResponse) error {
	if resp.Err != nil {
		return resp.Err
	}

	if len(resp.List) == 0 {
		return nil
	}

	for _, proxyIp := range resp.List {
		saveToDb(h.context.Db, h.Provider.GetSource(), proxyIp)
		saveToRedis(h.context.RedisClient, proxyIp)
	}

	return nil
}

func saveToDb(db *gorm.DB, source string, data ProxyIp) error {
	tmpProxy := model.HTTPProxyPool{}
	db.Where("source=? AND ip=? AND port=?", source, data.Ip, data.Port).First(&tmpProxy)
	if tmpProxy.ID == 0 {
		tmpProxy = model.HTTPProxyPool{
			Source:     source,
			IP:         data.Ip,
			Port:       strconv.Itoa(data.Port),
			City:       data.Addr,
			Isp:        data.Isp,
			ExpireTime: time.Now().Add(time.Second * time.Duration(data.Ttl)),
		}
	} else {
		tmpProxy.City = data.Addr
		tmpProxy.Isp = data.Isp
		tmpProxy.ExpireTime = time.Now().Add(time.Second * time.Duration(data.Ttl))
	}

	// save to db
	if err := db.Save(&tmpProxy).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"err":      err,
			"tmpProxy": tmpProxy,
		}).Error("save http proxy data fail.")
		return err
	}

	return nil
}

func saveToRedis(client *redis.Client, data ProxyIp) error {
	// save to client
	host := fmt.Sprintf("%v:%v", data.Ip, data.Port)
	expiredAt := time.Now().Unix() + int64(data.Ttl)

	client.ZAdd(CACHE_HTTP_PROXY_POOL_KEY, &redis.Z{
		Score:  float64(expiredAt),
		Member: host,
	})

	return nil
}

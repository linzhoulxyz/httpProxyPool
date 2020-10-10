package source

import (
	"fmt"
	"strconv"

	"github.com/samuncle-jqk/httpProxyPool/provider"

	"github.com/go-redis/redis/v7"
	"github.com/jinzhu/gorm"
	jsoniter "github.com/json-iterator/go"
	"github.com/samuncle-jqk/httpProxyPool/config"
	"github.com/samuncle-jqk/httpProxyPool/model"
	"github.com/samuncle-jqk/httpProxyPool/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type IpItem struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	ExpireTime string `json:"expire_time"`
	City       string `json:"city"`
	Isp        string `json:"isp"`
	Outip      string `json:"outip"`
}
type IpList struct {
	Code    int      `json:"code"`
	Data    []IpItem `json:"data"`
	Msg     string   `json:"msg"`
	Success bool     `json:"success"`
}

func reqAndSave() error {
	var (
		source = viper.GetString("proxy.source")
		preIp  = viper.GetInt("proxy.pre_ip")
		req    = fmt.Sprintf(viper.GetString("proxy.req_ip"), preIp)
	)
	db, err := config.GetDb()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("reqAndSave get db fail.")
		return err
	}

	redisClient, err := config.GetRedis()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("reqAndSave get redis fail.")
		return err
	}

	r := utils.NewRestyRequestChrome(nil)
	resp, err := r.Get(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("reqAndSave")
		return err
	}

	var rsp = IpList{}
	if err := jsoniter.Unmarshal(resp.Body(), &rsp); err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
			"rsp": rsp,
		}).Error("reqAndSave-unmarshal")
		return err
	}

	if rsp.Code != 0 {
		logrus.WithFields(logrus.Fields{
			"err": rsp.Msg,
			"rsp": rsp,
		}).Error("reqAndSave-unmarshal")
		return fmt.Errorf(rsp.Msg)
	}

	for _, data := range rsp.Data {
		saveToDb(db, source, data)
		saveToRedis(redisClient, data)
	}

	return nil
}

func saveToDb(db *gorm.DB, source string, data IpItem) error {
	tmpProxy := model.HTTPProxyPool{}
	db.Where("source=? AND ip=? AND port=?", source, data.IP, data.Port).First(&tmpProxy)
	if tmpProxy.ID == 0 {
		tmpProxy = model.HTTPProxyPool{
			Source:     source,
			IP:         data.IP,
			Port:       strconv.Itoa(data.Port),
			City:       data.City,
			Isp:        data.Isp,
			ExpireTime: utils.Str2Time(data.ExpireTime, ""),
			Outip:      data.Outip,
		}
	} else {
		tmpProxy.City = data.City
		tmpProxy.Isp = data.Isp
		tmpProxy.ExpireTime = utils.Str2Time(data.ExpireTime, "")
		tmpProxy.Outip = data.Outip
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

func saveToRedis(client *redis.Client, data IpItem) error {
	// save to client
	host := fmt.Sprintf("%v:%v", data.IP, data.Port)
	expiredAt := utils.Str2Time(data.ExpireTime, "").Unix()

	client.ZAdd(provider.CACHE_HTTP_PROXY_POOL_KEY, &redis.Z{
		Score:  float64(expiredAt),
		Member: host,
	})

	return nil
}

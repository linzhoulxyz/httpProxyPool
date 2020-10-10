package model

import "time"

// HTTPProxyPool [...]
type HTTPProxyPool struct {
	ID         int       `gorm:"primary_key;column:id;type:int(11);not null" json:"-"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:datetime;not null" json:"updated_at"`
	Source     string    `gorm:"unique_index:idx_ip;column:source;type:varchar(6);not null" json:"source"`
	IP         string    `gorm:"unique_index:idx_ip;column:ip;type:varchar(16);not null" json:"ip"`
	Port       string    `gorm:"unique_index:idx_ip;column:port;type:varchar(6);not null" json:"port"`
	City       string    `gorm:"column:city;type:varchar(24);not null" json:"city"`
	Isp        string    `gorm:"column:isp;type:varchar(12);not null" json:"isp"`
	ExpireTime time.Time `gorm:"index:idx_expire;column:expire_time;type:datetime;not null" json:"expire_time"`
	Outip      string    `gorm:"column:outip;type:varchar(16);not null" json:"outip"`
}

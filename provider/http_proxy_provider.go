package provider

type HttpProxyProvider interface {
	GetSource() string           // 代理源
	GetRequestInterval() int     // 请求间隔，单位秒
	GetProxyIp() ProxyIpResponse // 请求获取代理IP
	BindWhiteIp()                // 请求绑定IP白名单
}

type ProxyIp struct {
	Ip   string
	Port int
	Addr string
	Ttl  int // 存活秒数
	Isp  string
}

type ProxyIpResponse struct {
	Err  error
	List []ProxyIp
}

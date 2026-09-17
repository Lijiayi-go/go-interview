package main

import (
	"strconv"
	"strings"
)

// Asset 表示一个被识别出的 mDNS 资产。
// 至少包含 ip / port / host 三个维度，以及深度识别的 banner。
type Asset struct {
	IP       string            `json:"ip"`
	Port     int               `json:"port"`
	Host     string            `json:"host"`
	Protocol string            `json:"protocol"` // 服务协议名，如 http / smb / workstation / afpovertcp
	Service  string            `json:"service"`  // 完整服务类型，如 _http._tcp.local
	Name     string            `json:"name"`     // 实例名，如 slw-nas
	IPv6     string            `json:"ipv6,omitempty"`
	TTL      uint32            `json:"ttl"`
	Banner   map[string]string `json:"banner"` // 深度识别的 TXT key=value
	PTRs     []string          `json:"ptrs,omitempty"`
}

// ScanOptions 是扫描参数。
type ScanOptions struct {
	CIDR     string // 目标网段，如 192.168.1.0/24
	Ports    string // 端口范围，如 1-65535 或 80,443,5000-6000
	Timeout  int    // 单次查询超时（秒）
	Iface    string // 指定组播出接口（可选）
	Services string // 逗号分隔的服务类型（可选，默认常用类型）
}

// portSet 把端口范围字符串解析成可快速查询的集合。
type portSet struct {
	all   bool
	ports map[int]bool
}

func parsePortSet(s string) *portSet {
	ps := &portSet{ports: map[int]bool{}}
	s = strings.TrimSpace(s)
	if s == "" {
		ps.all = true
		return ps
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.IndexByte(part, '-'); i >= 0 {
			lo, _ := strconv.Atoi(strings.TrimSpace(part[:i]))
			hi, _ := strconv.Atoi(strings.TrimSpace(part[i+1:]))
			for p := lo; p <= hi && p <= 65535; p++ {
				ps.ports[p] = true
			}
		} else {
			p, _ := strconv.Atoi(part)
			ps.ports[p] = true
		}
	}
	return ps
}

func (p *portSet) contains(port int) bool {
	if p.all {
		return true
	}
	return p.ports[port]
}

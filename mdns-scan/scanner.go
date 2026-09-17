package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// 默认要枚举的服务类型。这些覆盖了常见的 mDNS 资产：
// 网络存储（NAS）、打印机、工作站、HTTP 服务等。
var defaultServices = []string{
	"_workstation._tcp.local.",
	"_http._tcp.local.",
	"_https._tcp.local.",
	"_smb._tcp.local.",
	"_afpovertcp._tcp.local.",
	"_device-info._tcp.local.",
	"_qdiscover._tcp.local.",
	"_ipp._tcp.local.",
	"_printer._tcp.local.",
	"_ssh._tcp.local.",
	"_ftp._tcp.local.",
	"_airplay._tcp.local.",
	"_googlecast._tcp.local.",
	"_hap._tcp.local.",
}

// parseServiceProtocol 从完整服务类型中提取协议名。
// 例如 "_afpovertcp._tcp.local." -> "afpovertcp"
func parseServiceProtocol(service string) string {
	s := strings.TrimPrefix(service, "_")
	// 去掉 ._tcp.local. / ._udp.local. 等后缀
	if i := strings.Index(s, "."); i > 0 {
		s = s[:i]
	}
	return s
}

// Scanner 执行 mDNS 资产测绘。
type Scanner struct {
	opts     ScanOptions
	ports    *portSet
	services []string
}

func NewScanner(opts ScanOptions) *Scanner {
	services := defaultServices
	if opts.Services != "" {
		services = nil
		for _, s := range strings.Split(opts.Services, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			// 补全后缀，方便用户只写 _http 或 http
			if !strings.HasSuffix(s, ".") {
				if !strings.HasPrefix(s, "_") {
					s = "_" + s
				}
				s = s + "._tcp.local."
			}
			services = append(services, s)
		}
	}
	return &Scanner{
		opts:     opts,
		ports:    parsePortSet(opts.Ports),
		services: services,
	}
}

// Scan 执行扫描，返回识别到的资产列表（已去重）。
func (s *Scanner) Scan(ctx context.Context) ([]Asset, error) {
	timeout := time.Duration(s.opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	// 从 CIDR 获取要绑定的源接口/地址，用于组播出口。
	ifaces, err := s.resolveInterfaces()
	if err != nil {
		return nil, fmt.Errorf("解析网段失败: %w", err)
	}

	var (
		mu     sync.Mutex
		assets = make(map[string]Asset)
		wg     sync.WaitGroup
	)

	// 对每个接口、每个服务类型并发发起组播查询。
	for _, iface := range ifaces {
		for _, svc := range s.services {
			wg.Add(1)
			go func(iface *net.Interface, svc string) {
				defer wg.Done()
				s.browseService(ctx, iface, svc, timeout, &mu, assets)
			}(iface, svc)
		}
	}
	wg.Wait()

	result := make([]Asset, 0, len(assets))
	for _, a := range assets {
		result = append(result, a)
	}
	return result, nil
}

// resolveInterfaces 根据 CIDR 解析出用于组播的接口列表。
func (s *Scanner) resolveInterfaces() ([]*net.Interface, error) {
	// 若用户指定了接口，直接用。
	if s.opts.Iface != "" {
		iface, err := net.InterfaceByName(s.opts.Iface)
		if err != nil {
			return nil, fmt.Errorf("找不到接口 %s: %w", s.opts.Iface, err)
		}
		return []*net.Interface{iface}, nil
	}

	// 如果没给 CIDR，返回所有开启组播的接口。
	if s.opts.CIDR == "" {
		ifaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}
		var out []*net.Interface
		for i := range ifaces {
			if ifaces[i].Flags&net.FlagMulticast != 0 && ifaces[i].Flags&net.FlagUp != 0 {
				out = append(out, &ifaces[i])
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("没有可用的组播接口")
		}
		return out, nil
	}

	ip, ipNet, err := net.ParseCIDR(s.opts.CIDR)
	if err != nil {
		return nil, fmt.Errorf("CIDR 格式错误 %q: %w", s.opts.CIDR, err)
	}
	_ = ip

	// 找到包含该网段地址的接口。
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []*net.Interface
	for i := range ifaces {
		if ifaces[i].Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := ifaces[i].Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ipn net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ipn = v.IP
			case *net.IPAddr:
				ipn = v.IP
			}
			if ipn != nil && ipNet.Contains(ipn) {
				out = append(out, &ifaces[i])
				break
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("本地没有接口属于网段 %s", s.opts.CIDR)
	}
	return out, nil
}

// browseService 枚举某个服务类型下的所有实例。
func (s *Scanner) browseService(ctx context.Context, iface *net.Interface, svc string, timeout time.Duration, mu *sync.Mutex, assets map[string]Asset) {
	resolver, err := zeroconf.NewResolver(zeroconf.SelectIfaces([]net.Interface{*iface}))
	if err != nil {
		log.Printf("[warn] 初始化 resolver 失败 (%s): %v", svc, err)
		return
	}

	entries := make(chan *zeroconf.ServiceEntry, 16)
	cctx, cancel := context.WithTimeout(ctx, timeout)

	err = resolver.Browse(cctx, svc, "local.", entries)
	if err != nil {
		log.Printf("[warn] 查询 %s 失败: %v", svc, err)
		cancel()
		return
	}

	// 收集期间的所有响应。
	for {
		select {
		case entry, ok := <-entries:
			if !ok {
				cancel()
				return
			}
			s.recordEntry(entry, mu, assets)
		case <-cctx.Done():
			cancel()
			return
		}
	}
}

// recordEntry 把一个 ServiceEntry 转成 Asset 并去重记录。
func (s *Scanner) recordEntry(entry *zeroconf.ServiceEntry, mu *sync.Mutex, assets map[string]Asset) {
	// 从 SRV 记录取端口和主机。
	if entry.Port <= 0 || entry.Port > 65535 {
		return
	}
	// 端口范围过滤：mDNS 的端口信息来自 SRV 记录，而非 TCP 扫描。
	if !s.ports.contains(entry.Port) {
		return
	}

	host := entry.HostName
	host = strings.TrimSuffix(host, ".")

	// 解析 IPv4 / IPv6。
	ipv4, ipv6 := "", ""
	for _, ip := range entry.AddrIPv4 {
		ipv4 = ip.String()
		break
	}
	for _, ip := range entry.AddrIPv6 {
		ipv6 = ip.String()
		break
	}

	// 深度识别：解析 TXT 记录为 key=value banner。
	banner := map[string]string{}
	for _, kv := range entry.Text {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			banner[kv[:i]] = kv[i+1:]
		} else {
			banner[kv] = ""
		}
	}

	// Service 字段形如 "_http._tcp."，去掉末尾点得到 "_http._tcp.local" 风格展示。
	service := strings.TrimSuffix(entry.Service, ".")

	asset := Asset{
		IP:       ipv4,
		Port:     entry.Port,
		Host:     host,
		Protocol: parseServiceProtocol(service),
		Service:  service,
		Name:     strings.TrimSuffix(entry.Instance, "."),
		IPv6:     ipv6,
		TTL:      entry.TTL,
		Banner:   banner,
	}

	// 去重键：协议 + IP + 端口 + 实例名。
	key := fmt.Sprintf("%s|%s|%d|%s", asset.Protocol, asset.IP, asset.Port, asset.Name)

	mu.Lock()
	defer mu.Unlock()
	assets[key] = asset
}

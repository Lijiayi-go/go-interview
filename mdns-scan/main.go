package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

func main() {
	var (
		cidr     = flag.String("cidr", "", "目标网段 (CIDR)，如 192.168.1.0/24；留空则扫描本机所有网段")
		ports    = flag.String("ports", "", "端口范围过滤，如 1-65535 或 80,443,5000-6000；留空不过滤")
		timeout  = flag.Int("timeout", 5, "单次 mDNS 查询超时（秒）")
		iface    = flag.String("iface", "", "指定组播出接口名（可选）")
		services = flag.String("services", "", "逗号分隔的服务类型（可选，默认枚举常见类型）")
		jsonOut  = flag.Bool("json", false, "以 JSON 格式输出")
	)
	flag.Usage = usage
	flag.Parse()

	opts := ScanOptions{
		CIDR:     *cidr,
		Ports:    *ports,
		Timeout:  *timeout,
		Iface:    *iface,
		Services: *services,
	}

	ctx := context.Background()
	scanner := NewScanner(opts)

	log.Printf("开始 mDNS 资产测绘: cidr=%q ports=%q timeout=%ds", opts.CIDR, opts.Ports, opts.Timeout)
	assets, err := scanner.Scan(ctx)
	if err != nil {
		log.Fatalf("扫描失败: %v", err)
	}

	// 排序，输出稳定。
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].IP != assets[j].IP {
			return assets[i].IP < assets[j].IP
		}
		if assets[i].Port != assets[j].Port {
			return assets[i].Port < assets[j].Port
		}
		return assets[i].Protocol < assets[j].Protocol
	})

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(assets)
		return
	}

	printText(assets)
	log.Printf("完成，共识别 %d 个 mDNS 资产", len(assets))
}

// printText 以人类可读格式输出，贴近题目示例的 banner 深度。
func printText(assets []Asset) {
	if len(assets) == 0 {
		fmt.Println("未发现 mDNS 资产。")
		return
	}

	// 按 IP+Host 分组，同一台设备的多服务聚合展示。
	grouped := map[string][]Asset{}
	var order []string
	for _, a := range assets {
		key := a.Host
		if key == "" {
			key = a.IP
		}
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], a)
	}

	for _, key := range order {
		list := grouped[key]
		host := list[0].Host
		ip := list[0].IP
		ipv6 := list[0].IPv6
		fmt.Printf("\n%s (%s):\n", host, ip)
		if ipv6 != "" {
			fmt.Printf("  IPv6=%s\n", ipv6)
		}
		for _, a := range list {
			fmt.Printf("  %d/tcp %s:\n", a.Port, a.Protocol)
			fmt.Printf("    Name=%s\n", a.Name)
			if a.IP != "" {
				fmt.Printf("    IPv4=%s\n", a.IP)
			}
			if a.IPv6 != "" {
				fmt.Printf("    IPv6=%s\n", a.IPv6)
			}
			fmt.Printf("    Hostname=%s\n", a.Host)
			fmt.Printf("    TTL=%d\n", a.TTL)
			// 深度 banner：TXT 的 key=value
			if len(a.Banner) > 0 {
				keys := make([]string, 0, len(a.Banner))
				for k := range a.Banner {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				pairs := make([]string, 0, len(keys))
				for _, k := range keys {
					if a.Banner[k] == "" {
						pairs = append(pairs, k)
					} else {
						pairs = append(pairs, k+"="+a.Banner[k])
					}
				}
				fmt.Printf("    %s\n", strings.Join(pairs, ","))
			}
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `mdns-scan - mDNS 资产测绘 CLI

用法:
  mdns-scan [flags]

参数:
  -cidr string      目标网段 (CIDR)，如 192.168.1.0/24；留空扫描本机所有网段
  -ports string     端口范围过滤，如 1-65535 或 80,443,5000-6000；留空不过滤
  -timeout int      单次 mDNS 查询超时（秒，默认 5）
  -iface string     指定组播出接口名（可选）
  -services string  逗号分隔的服务类型（可选）
  -json             以 JSON 格式输出

示例:
  mdns-scan -cidr 192.168.1.0/24 -ports 1-65535
  mdns-scan -cidr 10.0.0.0/16 -services _http,_smb -json
`)
}

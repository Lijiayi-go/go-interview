# mdns-scan

一个基于 Golang 的 **mDNS 资产测绘 CLI** 工具。输入 IP 网段和端口范围，通过 Multicast DNS（RFC 6762）组播协议发现局域网内的设备与服务，输出包含 `ip` / `port` / `host` 及**深度识别的 banner**（TXT 记录 key=value、服务类型等）。

## 工作原理

mDNS 使用组播地址 `224.0.0.251:5353`（IPv4）进行服务发现，端口信息来自 SRV 记录而非 TCP SYN 扫描。工具向目标网段发送 PTR 查询，解析返回的 DNS 记录：

- **PTR** → 服务实例名
- **SRV** → 目标主机 + 端口
- **TXT** → banner（`key=value` 字段，如 `model=TS-X64`、`fwVer=5.2.9`）
- **A / AAAA** → IPv4 / IPv6 地址

## 构建

```bash
# 本地编译
go build -o mdns-scan .

# Docker 构建
docker build -t mdns-scan .
```

## 运行

```bash
# 本地运行：扫描 192.168.1.0/24 网段，端口不限
./mdns-scan -cidr 192.168.1.0/24 -ports 1-65535

# 扫描所有网段，只关注 80、443、5000-6000 端口
./mdns-scan -ports 80,443,5000-6000 -timeout 8

# 指定服务类型，JSON 输出
./mdns-scan -cidr 10.0.0.0/16 -services _http,_smb -json

# Docker 运行（组播需 host 网络）
docker run --rm --network host mdns-scan -cidr 192.168.1.0/24
```

> **注意**：mDNS 依赖 UDP 组播，Docker 默认 bridge 网络无法转发组播包，必须使用 `--network host`（Linux）或 `--cap-add=NET_ADMIN`。

## 参数

| 参数 | 说明 | 默认 |
|------|------|------|
| `-cidr` | 目标网段（CIDR），如 `192.168.1.0/24` | 空（扫描本机所有网段） |
| `-ports` | 端口范围过滤，如 `1-65535` 或 `80,443,5000-6000` | 空（不过滤） |
| `-timeout` | 单次 mDNS 查询超时（秒） | 5 |
| `-iface` | 指定组播出接口名 | 空（自动选择） |
| `-services` | 逗号分隔的服务类型，如 `_http,_smb` | 常见类型 |
| `-json` | 以 JSON 格式输出 | false |

## 输出示例

```text
slw-nas (192.168.1.100):
  IPv6=fe80::265e:beff:fe69:a313
  9/tcp workstation:
    Name=slw-nas [24:5e:be:69:a3:13]
    IPv4=192.168.1.100
    IPv6=fe80::265e:beff:fe69:a313
    Hostname=slw-nas.local
    TTL=10
  5000/tcp http:
    Name=slw-nas
    IPv4=192.168.1.100
    Hostname=slw-nas.local
    TTL=10
    path=/
  5000/tcp qdiscover:
    Name=slw-nas
    IPv4=192.168.1.100
    Hostname=slw-nas.local
    TTL=10
    accessType=https,accessPort=86,model=TS-X64,displayModel=TS-464C,fwVer=5.2.9,fwBuildNum=20260214
  548/tcp afpovertcp:
    Name=slw-nas(AFP)
    IPv4=192.168.1.100
    Hostname=slw-nas.local
    TTL=10
    model=Xserve
```

## 项目结构

```
mdns-scan/
├── main.go       # CLI 入口与输出格式化
├── scanner.go    # mDNS 组播扫描核心逻辑
├── model.go      # 数据结构与参数解析
├── Dockerfile    # 多阶段构建
└── README.md
```

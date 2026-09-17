# mdns-scan 输出样例

以下为对 `192.168.1.0/24` 网段执行
`mdns-scan -cidr 192.168.1.0/24 -ports 1-65535` 的期望输出（示例数据）。

## 文本输出

```
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
    IPv6=fe80::265e:beff:fe69:a313
    Hostname=slw-nas.local
    TTL=10
    path=/
  445/tcp smb:
    Name=slw-nas
    IPv4=192.168.1.100
    IPv6=fe80::265e:beff:fe69:a313
    Hostname=slw-nas.local
    TTL=10
  5000/tcp qdiscover:
    Name=slw-nas
    IPv4=192.168.1.100
    IPv6=fe80::265e:beff:fe69:a313
    Hostname=slw-nas.local
    TTL=10
    accessType=https,accessPort=86,model=TS-X64,displayModel=TS-464C,fwVer=5.2.9,fwBuildNum=20260214
  548/tcp afpovertcp:
    Name=slw-nas(AFP)
    IPv4=192.168.1.100
    IPv6=fe80::265e:beff:fe69:a313
    Hostname=slw-nas.local
    TTL=10
    model=Xserve
```

## JSON 输出

```json
[
  {
    "ip": "192.168.1.100",
    "port": 5000,
    "host": "slw-nas.local",
    "protocol": "qdiscover",
    "service": "_qdiscover._tcp",
    "name": "slw-nas",
    "ipv6": "fe80::265e:beff:fe69:a313",
    "ttl": 10,
    "banner": {
      "accessType": "https",
      "accessPort": "86",
      "model": "TS-X64",
      "displayModel": "TS-464C",
      "fwVer": "5.2.9",
      "fwBuildNum": "20260214"
    }
  }
]
```

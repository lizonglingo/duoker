package network

import (
	"duoker/config"
	"duoker/log"
	"encoding/binary"
	"encoding/json"
	"net"
	"os"
)

// ipAmFs IP 分配管理.
// IP 地址管理 (IPAM) 是一套集成工具
// 支持端到端规划、部署、管理和监视 IP 地址基础结构
// 同时提供丰富的用户体验。
// IPAM 自动发现网络上的 IP 地址基础结构服务器和域名系统 (DNS) 服务器，使用户能够从中心界面管理它们。
type ipAmFs struct {
	subnets map[string]*bitMap
	path    string
}

// IpAmfs IpAmfs 保存网络地址分配的结构体信息和Path信息
var IpAmfs = &ipAmFs{
	subnets: make(map[string]*bitMap),
	path:    config.IpAmStorageFsPath,
}

func (ipamfs *ipAmFs) SetIpUsed(subnet string) error {
	if err := ipamfs.loadConf(); err != nil {
		return err
	}
	// 从 CIDR 中解析网络信息
	ip, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}
	// IP 转为 IPv4
	ip = ip.To4()
	ones, total := cidr.Mask.Size()
	// 每个子网映射有一个 bitmap 来存储这个子网中 IP 的使用情况
	bitmap := ipamfs.subnets[cidr.String()]
	// 如果没有就进行初始化
	if bitmap == nil || bitmap.Bitmap == nil {
		bitmap = InitBitMap(1 << (total - ones))
		ipamfs.subnets[cidr.String()] = bitmap
	}
	// 获得这个 IP 在子网中的位置 - 索引
	pos := getIPIndex(ip, cidr.Mask)
	log.Debug("set ip %s pos %d \n", ip, pos)
	// 然后在 bitmap 设置为对应的值
	bitmap.BitSet(pos)
	return ipamfs.sync()
}

// AllocIp 遍历 bitmap 寻找还没有使用的 IP 号
// 然后进行分配.
func (ipamfs *ipAmFs) AllocIp(subnet string) (net.IP, error) {
	if err := ipamfs.loadConf(); err != nil {
		return nil, err
	}
	ip, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}
	ip = ip.To4()
	ones, total := cidr.Mask.Size()
	bitmap := ipamfs.subnets[cidr.String()]
	if bitmap == nil || bitmap.Bitmap == nil {
		bitmap = InitBitMap(1 << (total - ones))
		ipamfs.subnets[cidr.String()] = bitmap
	}

	// pos 为 0 是网络号不能分配ip，
	for pos := 1; pos <= (1<<(total-ones) - 2); pos++ {
		if bitmap.BitExist(pos) {
			continue
		}
		bitmap.BitSet(pos)
		firstIP := ipToUint32(ip.Mask(cidr.Mask))
		ip = uint32ToIP(firstIP + uint32(pos))
		break
	}
	err = ipamfs.sync()
	if err != nil {
		return nil, err
	}
	return ip, nil
}

// ReleaseIp 根据 IP 在子网中的索引 清除这个 IP 的使用记录.
func (ipamfs *ipAmFs) ReleaseIp(subnet string, ip net.IP) error {
	if err := ipamfs.loadConf(); err != nil {
		return err
	}
	_, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}
	bitmap := ipamfs.subnets[cidr.String()]
	if bitmap == nil {
		return nil
	}
	pos := getIPIndex(ip, cidr.Mask)
	bitmap.BitClean(pos)
	return ipamfs.sync()
}

func uint32ToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// getIPIndex 获取某个 IP 地址在对应 CIDR 子网中的索引/顺序
// (该子网中第 Index 个 IP).
func getIPIndex(ip net.IP, mask net.IPMask) int {
	ipInt := ipToUint32(ip)
	firstIP := ipToUint32(ip.Mask(mask))
	return int(ipInt - firstIP)
}
func ipToUint32(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

// loadConf 从持久化的文件中加载 ipam 的数据
// 解析到内存的结构体中.
func (ipamfs *ipAmFs) loadConf() error {
	if _, err := os.Stat(ipamfs.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	data, err := os.ReadFile(ipamfs.path)
	if err != nil {
		return err
	}
	if len(ipamfs.subnets) == 0 {
		ipamfs.subnets = make(map[string]*bitMap)
	}
	if len(data) == 0 {
		return nil
	}
	err = json.Unmarshal(data, &ipamfs.subnets)
	if err != nil {
		return err
	}
	return nil
}

// sync 将 IP 分配的信息写到持久化文件中.
func (ipamfs *ipAmFs) sync() error {
	if _, err := os.Stat(ipamfs.path); err != nil {
		if os.IsNotExist(err) {
			os.Create(ipamfs.path)
		} else {
			return err
		}
	}
	data, err := json.Marshal(ipamfs.subnets)
	if err != nil {
		return err
	}
	err = os.WriteFile(ipamfs.path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

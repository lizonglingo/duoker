// Package network 处理容器网络的主要逻辑
// 包括 创建/初始化网桥设备 分配容器网络 IP
// 清理网络设备 回收 IP 等
// bridge_network 构建配置网桥
// ipam_fs 用来管理 IP 地址的分配和回收
// bitmap 用于 IP 地址分配记录.
package network

import (
	"duoker/config"
	"duoker/log"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// NetConf 网络配置信息.
type NetConf struct {
	NetworkName string     // 网络名称
	IpRange     *net.IPNet // IP 地址范围
	Driver      string     // 驱动
	BridgeName  string     // 网桥名称
	BridgeIp    *net.IPNet // 网桥的 IP
}

// netMgr 用于存储网络配置信息.
type netMgr struct {
	Storage map[string]*NetConf
}

// NetMgr 初始化
var NetMgr = &netMgr{
	Storage: map[string]*NetConf{},
}

// Sync 将 netMgr 的信息写道文件中
// 实现一个简单的持久化存储.
func (n *netMgr) Sync() error {
	// 判断网络信息存储的文件是否存在
	if _, err := os.Stat(config.NetStoragePath); err != nil {
		// 有点像 kubernetes cni 中的 cilium-05 flannel-10 这种文件
		if os.IsNotExist(err) {
			// 不存在就创建这个文件
			// 在宿主机的这个文件中 /workplace/duoker/netconfig/network.json
			os.Create(config.NetStoragePath)
		} else {
			return err
		}
	}
	// 序列化
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	// 将序列化的网络信息写入文件中
	err = os.WriteFile(config.NetStoragePath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// LoadConf 从文件中读取网络配置.
func (n *netMgr) LoadConf() error {
	if _, err := os.Stat(config.NetStoragePath); err != nil {
		if os.IsNotExist(err) {
			log.Info("cannot found network config file in path %s", config.NetStoragePath)
			return nil
		} else {
			return err
		}
	}

	// 将文件中的数据反序列化到结构体中
	data, err := os.ReadFile(config.NetStoragePath)
	if err != nil {
		return err
	}
	if len(n.Storage) == 0 {
		n.Storage = make(map[string]*NetConf)
	}
	if len(data) == 0 {
		return nil
	}
	err = json.Unmarshal(data, n)
	if err != nil {
		return err
	}
	return nil
}

const (
	defaultNetName = "duoker-bridge"  // 默认的网桥名称
	defaultSubnet  = "192.169.0.1/24" // 默认的子网地址
)

type networktype string

const (
	// BridgeNetworkType 目前只支持网桥类型的网络设备
	BridgeNetworkType networktype = "bridge"
)

// String 实现 string 接口.
func (n networktype) String() string {
	return string(n)
}

// Init 初始化默认的容器网络.
func Init() error {
	// 对默认网络进行初始化
	if err := BridgeDriver.CreateNetwork(defaultNetName, defaultSubnet, BridgeNetworkType); err != nil {
		return fmt.Errorf("err=%s", err)
	}
	if err := IpAmfs.SetIpUsed(defaultSubnet); err != nil {
		return err
	}
	return nil
}

// ConfigDefaultNetworkInNewNet 配置网络命名空间
// 配置 veth对 将容器中的网络和宿主机的网络连在一起.
func ConfigDefaultNetworkInNewNet(pid int) error {
	// 为 veth 分配新的 IP
	ip, err := IpAmfs.AllocIp(defaultSubnet)
	if err != nil {
		return fmt.Errorf("ipam alloc ip fail %s", err)
	}

	// 主机上创建 veth 设备,并连接到网桥上
	vethLink, networkConf, err := BridgeDriver.CrateVeth(defaultNetName)
	if err != nil {
		return fmt.Errorf("create veth fail err=%s", err)
	}
	// 主机上设置子进程网络命名空间 配置
	if err := BridgeDriver.setContainerIp(vethLink.PeerName, pid, ip, networkConf.BridgeIp); err != nil {
		return fmt.Errorf("setContainerIp fail err=%s peername=%s pid=%d ip=%v conf=%+v", err, vethLink.PeerName, pid, ip, networkConf)
	}
	// 通知子进程设置完毕
	log.Debug("parent process set ip success")
	return noticeSunProcessNetConfigFin(pid)
}

func noticeSunProcessNetConfigFin(pid int) error {
	return syscall.Kill(pid, syscall.SIGUSR2)
}

func WaitParentSetNewNet() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)
	<-sigs
	log.Info("Received SIGUSR2 signal, prepare run container")
}

package network

import (
	"duoker/log"
	"fmt"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type bridgeDriver struct {
}

func (b *bridgeDriver) Name() string {
	return "bridge"
}

var BridgeDriver = &bridgeDriver{}

// truncate 截断超过长度的.
func truncate(maxlen int, str string) string {
	if len(str) <= maxlen {
		return str
	}
	return str[:maxlen]
}

// createBridge 为容器创建 宿主机上的 bridge 网桥设备.
// type IPNet struct {
//     IP   IP
//     Mask IPMask
// }
func createBridge(networkName string, interfaceIP *net.IPNet) (string, error) {
	// 构造网桥名称
	bridgeName := truncate(15, fmt.Sprintf("br-%s", networkName))
	// NewLinkAttrs 初始网络设备的属性
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	br := &netlink.Bridge{LinkAttrs: la}
	// Equivalent to: `ip link add $link`
	// 加入设备
	if err := netlink.LinkAdd(br); err != nil {
		return "", fmt.Errorf("bridge creation failed for bridge %s: %s", bridgeName, err)
	}
	// 配置设备地址
	addr := &netlink.Addr{IPNet: interfaceIP, Peer: interfaceIP, Label: "", Flags: 0, Scope: 0}
	if err := netlink.AddrAdd(br, addr); err != nil {
		return "", fmt.Errorf("bridge add addr fail %s", err)
	}

	// `ip link set $link up`
	// 启用设备
	if err := netlink.LinkSetUp(br); err != nil {
		return "", fmt.Errorf("error enabling interface for %s: %v", bridgeName, err)
	}

	return bridgeName, nil
}

// genInterfaceIp 生成网络接口的 IP 地址.
func genInterfaceIp(rawIpWithRange string) (*net.IPNet, error) {
	ipNet, err := netlink.ParseIPNet(rawIpWithRange)
	if err != nil {
		return nil, fmt.Errorf("parse ip fail ip=%+s err=%s", rawIpWithRange, err)
	}
	return ipNet, nil
}

// setSNat 为 iptables 配置 NAT.
func setSNat(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("set snat fail %s", err)
	}
	return nil
}

// CreateNetwork 创建网络.
func (b *bridgeDriver) CreateNetwork(networkName string, subnet string, networkType networktype) error {
	// 当前仅处理宿主机的桥接网络
	if networkType != BridgeNetworkType {
		return fmt.Errorf("support bridge network type now ")
	}

	// 检查网络命名是否存在
	// 从持久化的文件中加载数据
	if err := NetMgr.LoadConf(); err != nil {
		return fmt.Errorf("netMgr loadConf fail %s", err)
	}
	// 查看是否已经存在这个网桥设备
	if netConf, ok := NetMgr.Storage[networkName]; ok {
		switch netConf.Driver {
		case "bridge":
			// 系统重启后需要重新建立网桥配置
			_, err := netlink.LinkByName(netConf.BridgeName)
			// 如果存在就直接返回不用进行后面的创建操作了
			if err == nil {
				log.Info("exist default network ,will not create new network ")
				return nil
			}
		default:
			return fmt.Errorf("not support network driver")
		}
	}

	// 如果之前没有创建过这个网络设备
	// 就重新创建网桥
	interfaceIp, err := genInterfaceIp(subnet)
	if err != nil {
		return fmt.Errorf("genInterfaceIp err=%s", err)
	}
	bridgeName, err := createBridge(networkName, interfaceIp)
	if err != nil {
		return fmt.Errorf("createBridge err=%s", err)
	}

	// 从给定的子网地址 CIDR 中解析出 IP SUBNET 等信息
	// 192.0.2.1
	// 192.0.2.0/24
	// 2001:db8:a0b:12f0::1
	// 2001:db8::/32
	// return  (IP, *IPNet, error)
	_, cidr, _ := net.ParseCIDR(subnet)

	// 根据解析的子网信息为宿主机配置 NAT
	err = setSNat(bridgeName, cidr)
	if err != nil {
		log.Error("%s", err)
	}
	// 最后进行记录
	NetMgr.Storage[networkName] = &NetConf{
		NetworkName: networkName,
		IpRange:     cidr,
		Driver:      BridgeNetworkType.String(),
		BridgeName:  bridgeName,
		BridgeIp:    interfaceIp,
	}
	return NetMgr.Sync()
}

// CrateVeth 创建 veth 设备连接容器和宿主机的网络.
func (b *bridgeDriver) CrateVeth(networkName string) (*netlink.Veth, *NetConf, error) {
	// 检查网络命名是否存在
	// 要先检查 Veth 所在的网络是否被正常配置
	if err := NetMgr.LoadConf(); err != nil {
		return nil, nil, fmt.Errorf("netMgr loadConf fail %s", err)
	}
	networkConf, ok := NetMgr.Storage[networkName]
	if !ok {
		return nil, nil, fmt.Errorf("name %s network is invalid", networkName)
	}

	// 找到网络对应的 link
	br, err := netlink.LinkByName(networkConf.BridgeName)
	if err != nil {
		return nil, nil, fmt.Errorf("link by name fail err=%s", err)
	}
	// 获得默认的属性 例如：
	//  MTU          int
	//	TxQLen       int // Transmit Queue Length
	//	Name         string
	//	Flags        net.Flags
	//	RawFlags     uint32
	//	Namespace    interface{} // nil | NsPid | NsFd
	//	Xdp          *LinkXdp
	//	NetNsID      int
	//	NumTxQueues  int
	//	NumRxQueues  int
	// 等
	la := netlink.NewLinkAttrs()
	vethname := truncate(15, "veth-"+strconv.Itoa(10+int(rand.Int31n(10)))+"-"+networkConf.NetworkName)
	la.Name = vethname
	la.MasterIndex = br.Attrs().Index
	// 创建 veth 设备
	vethLink := &netlink.Veth{
		LinkAttrs: la,
		PeerName:  truncate(15, "cif-"+vethname),
	}
	// `ip link add $link`
	if err := netlink.LinkAdd(vethLink); err != nil {
		return nil, nil, fmt.Errorf("veth creation failed for bridge %s: %s", networkName, err)
	}
	//  `ip link set $link up`
	if err := netlink.LinkSetUp(vethLink); err != nil {
		return nil, nil, fmt.Errorf("error enabling interface for %s: %v", networkName, err)
	}
	return vethLink, networkConf, nil
}

func (b *bridgeDriver) setContainerIp(peerName string, pid int, containerIp net.IP, gateway *net.IPNet) error {
	peerLink, err := netlink.LinkByName(peerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	loLink, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	// 进入容器的网络命名空间
	defer enterContainerNetns(&peerLink, pid)()
	containerVethInterfaceIP := *gateway
	containerVethInterfaceIP.IP = containerIp
	if err = setInterfaceIP(peerName, containerVethInterfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", containerIp, err)
	}
	if err := netlink.LinkSetUp(peerLink); err != nil {
		return fmt.Errorf("netlink.LinkSetUp fail  name=%s err=%s", peerName, err)
	}
	if err := netlink.LinkSetUp(loLink); err != nil {
		return fmt.Errorf("netlink.LinkSetUp fail  name=%s err=%s", peerName, err)
	}
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        gateway.IP,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("router add fail %s", err)
	}

	return nil
}

// enterContainerNetns 进入容器的网络命名空间中.
func enterContainerNetns(vethLink *netlink.Link, pid int) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/ns/net", pid), os.O_RDONLY, 0)
	if err != nil {
		fmt.Println(fmt.Errorf("error get container net namespace, %v", err))
	}

	// 拿到文件描述符
	nsFD := f.Fd()
	// 对线程加锁
	// 这里和 golang 的协程调度模型有关
	// G(goroutine) M(machine) P(process)
	// LockOSThread 锁定当前下协程
	runtime.LockOSThread()

	// 修改 veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*vethLink, int(nsFD)); err != nil {
		log.Error("error set link netns , %v", err)
	}

	// 获取当前的网络 namespace
	origns, err := netns.Get()
	if err != nil {
		log.Error("error get current netns, %v", err)
	}

	// 设置当前线程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Error("error set netns, %v", err)
	}

	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

// Set the IP addr of a netlink interface
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		fmt.Println(fmt.Errorf("error retrieving new bridge netlink link [ %s ]... retrying", name))
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("Abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{IPNet: ipNet, Peer: ipNet, Label: "", Flags: 0, Scope: 0}
	fmt.Println("veth peer ip: ", ipNet.IP)
	return netlink.AddrAdd(iface, addr)
}

package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/vishvananda/netlink"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

type BridgeNetWorkDriver struct {
}

// Name 是NetworkDriver接口的方法, 返回驱动的名称
func (d *BridgeNetWorkDriver) Name() string {
	return "bridge"
}

// Create 是NetworkDriver接口的方法,创建bridge网络驱动, 主要是用过initBridge方法来初始化网桥的
func (d *BridgeNetWorkDriver) Create(subnet string, name string) (*Network, error) {

	//通过ParseCIDR获取网段的字符串中的网关IP地址和网络IP段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip

	//初始化网络对象
	n := &Network{
		Name:    name,
		IpRange: ipRange,
	}

	//配置Linux bridge
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("Error init birdge : %v ", err)
	}
	return n, err
}

// Delete 是NetworkDriver接口的方法,删除network对应的网桥设备
func (d *BridgeNetWorkDriver) Delete(network Network) error {
	//获取网桥设备的名字
	bridgeName := network.Name
	//通过设备名找到网络对应的
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

//Connect 是NetworkDriver接口的方法,使得网桥和veth相连接. br是网桥设备,配置端点设备即veth的相关信息
func (d *BridgeNetWorkDriver) Connect(network *Network, endpoint *Endpoint) error {

	//获取网络名,即Linux Bridge的接口名
	bridgeName := network.Name
	//通过接口名获取接口的对象和接口的属性
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	//创建veth接口的配置
	la := netlink.NewLinkAttrs()
	//由于Linux接口名的限制,取前5位
	la.Name = endpoint.ID[:5]

	//通过设置veth接口的master属性,设置这个veth的一端挂载到网络对应的Linux Bridge上
	la.MasterIndex = br.Attrs().Index

	//创建veth设备,同时创建的还有peer veth
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	//LinkAdd 创建Veth接口,因为la.MasterIndex = br.Attrs().Index 所有veth的另一端已经挂载到网络对应的Linux Bridge上
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device :%v", err)
	}

	//启动endpoint设备
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Set UP Endpoing Device : %v", err)
	}
	return nil
}

//Disconnect 是NetworkDriver接口的方法
func (d *BridgeNetWorkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

//initBridge 完成Linux中搭建Bridge流程
func (d *BridgeNetWorkDriver) initBridge(n *Network) error {

	//1.初始化Bridge设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error create Bridge dev interface %s ,Error : %v", bridgeName, err)
	}

	//2.设置Bridge设备的地址和路由
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP

	if err := setInteraceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("Error assigning address : %s on bridge : %s with an error of : %v", gatewayIP, bridgeName, err)
	}

	//3.启动Bride设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up : %s .Error :%v", bridgeName, err)
	}

	//4.设置iptables的SNAT规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s . Error : %v", bridgeName, err)
	}
	return nil

}

//1.创建Liunx Bridge设备
func createBridgeInterface(bridgeName string) error {

	//先检查是否存在这个同名的bridge设备
	_, err := net.InterfaceByName(bridgeName)

	//如果已经存在或则报错则返回创建错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	//初始化一个netlink的Link基础对象,Link的名字就是bridge虚拟设备的名字
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	//使用刚刚创建的Link的属性创建netlink的bridge对象
	br := &netlink.Bridge{la}

	//调用netlink的Linkadd ,创建bridge虚拟网络设备,相当于ip link add xxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge creation failed for bridge %s : %v", bridgeName, err)
	}
	return nil

}

// 2.设置Bridge设备的地址和路由 ,不仅限于Bridge,也可以对容器内的veth进行操作
func setInteraceIP(name string, rawIP string) error {
	//找到需要设置的网络接口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("Error get interface :%v", err)
	}
	//ipNet中既包含了网段信息192.168.0.0/24  也包含了原始的IP 192.168.0.1
	//netlink.ParseIPNet 是对ParseCIDR的一个封装,可同时返回上述信息
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}

	//AddrAdd给网络接口配置地址,Equivalent to: `ip addr add $addr dev $link`
	addr := &netlink.Addr{ipNet, "", 0, 0, nil} //构造传入的参数
	//给iface这个对象,配置了路由表,地址所在网段的信息,地址
	return netlink.AddrAdd(iface, addr)
}

//3.设置网络接口为UP状态,这时可以处理和转发请求
func setInterfaceUP(interfaceName string) error {
	//找到需要设置的网络接口
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ] : %v", iface.Attrs().Name, err)
	}

	//设置接口的状态为UP,Equivalent to: `ip link set $link up`
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s :%v", interfaceName, err)
	}
	return nil
}

/*
 4.设置iptables的SNAT规则,只要从这个网桥上传出来的包,都会对其做源IP的转换,保证容器经过宿主机访问到宿主机外部的网络请求的包转换成机器IP,
	简而言之就是可以让容器发送网络请求到宿主机外,这时容器可以正确的送到和就收宿主机外部的网络请求
	GO中没有操作iptables的库,需要手动添加指令
*/
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//执行iptables命令配置的SNAT规则
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}

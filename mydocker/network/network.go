package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/vishvananda/netns"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/vishvananda/netlink"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network/" //默认网络文件存储地址
	drivers            = map[string]NetworkDriver{}           //驱动
	networks           = map[string]*Network{}                //网络
)

// Network 网络相关配置
type Network struct {
	Name    string     //网络名
	IpRange *net.IPNet // 地址段
	Driver  string     //网络驱动名
}

// Endpoint 网络端点,相当于veth ,记录连接到网络的一些信息
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping`
	Network     *Network
}

//NetworkDriver 网络驱动,在启动是通过制定不同的网络驱动来使用哪个驱动做网络配置
type NetworkDriver interface {
	//驱动名
	Name() string
	//创建网络
	Create(subnet string, name string) (*Network, error)
	//删除网络
	Delete(network Network) error
	//连接容器网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	//从网络上移除容器网络端点
	Disconnect(network Network, endpoint *Endpoint) error
}

//CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	//将subnet转换成net.IPNet对象
	_, cidr, _ := net.ParseCIDR(subnet)

	//通过IPAM分配网关IP,获取网段中的第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp

	//调用制定的网络驱动创建网络,drivers字典是各个网络驱动的实例字典,通过调用网络驱动的Create方法创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	// 保存网络信息,将网络信息保存在文件系统中,方便查询和在网络上连接网络端点
	logrus.Infof("CreateNetwork driver name %s", nw.Name)
	return nw.dump(defaultNetworkPath)
}

//dump 将这个网络的配置信息保存在文件系统中
/*
	1. 创建路径
	2. 创建打开文件
	3. 网络信息json化
	4. json化的网络信息存入文件

*/
func (nw *Network) dump(dumpPath string) error {

	//检查保存的目录是否存在
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}

	nwPath := path.Join(dumpPath, nw.Name)

	//打开保存的文件用于写入,后面的参数模式依次是:存在内容则清空,只写入,不存在则创建
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("OpenFile nw error %v", err)
		return err
	}
	defer nwFile.Close()

	//通过json的库序列化网络对象到json的字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("Json nw error %v", err)
		return err
	}

	//将网络配置的json格式写入到文件中
	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("Write nwJson error %v", err)
		return err
	}

	return nil

}

//从网络的配置目录中的文件读取到 网络的配置
func (nw *Network) load(dumpPath string) error {
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}

	//从配置文件中读取网络的配置json字符串
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	//通过json字字符串反序列化出网络信息
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("Load nwfile error %v", err)
		return err
	}
	return nil

}

// Remove 删除网络配置目录中的文件,即配置目录下的网络名文件
func (nw *Network) Remove(dumpPath string) error {

	//检查文件状态,如果文件不存在直接返回
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		//删除文件
		return os.Remove(path.Join(dumpPath, nw.Name))
	}

}

// Connect 连接容器到之前创建的网络中 mydocker run -net testnet -p 8080:80 xxx
func Connect(networkName string, cinfo *container.ContainerInfo) error {

	//从networks字典中取到容器连接的网络信息,networks字典中保存当前以创建的网络
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such Network :%s ", networkName)
	}

	//通过调用IPAM,从网络的IP段中,分配容器IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	//创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName), //取容器ID的前5位+网络名,作为endpoint的ID
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}

	// 调用网络对应的网络驱动去挂载, 配置网络端点
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	//进入到容器的namespace配置容器网络,设备的IP地址和路由
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	//配置容器到宿主机的端口映射
	return configPortMapping(ep, cinfo)

}

//Init 初始化网络驱动
func Init() error {
	//加载网络驱动
	var brdgeDriver = BridgeNetWorkDriver{}
	drivers[brdgeDriver.Name()] = &brdgeDriver

	logrus.Infof("BridgeDrive Name : %s", brdgeDriver.Name())
	//判断网络的配置目录是否存在,不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	//检查网络配置目录中的所有文件,并对其做相应的处理
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {

		//如果是目录则跳过
		if info.IsDir() {
			return nil
		}

		//?加载文件名作为网络名
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		//调用load加载网络的配置信息
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("Error Load network : %s", err)
		}

		//将网络配置信息加入到networks字典中
		networks[nwName] = nw
		return nil
	})
	return nil
}

//ListNetwork 展示网络信息
func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")

	//遍历网络信息
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}

	//输出到标准输出
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

//DeleteNetwork 删除存在的网络
func DeleteNetwork(networkName string) error {

	//查找网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Newwork :%s", networkName)
	}

	//调用IPAM的实例ipAllocator释放网络网关的IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("Error Remove Network gatewayip : %s", err)
	}

	//调用网络驱动删除网络创建的设备与配置
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error Remove Network Driver %s", err)
	}

	//从网络的配置目录中删除该网络对应的配置文件
	return nw.Remove(defaultNetworkPath)

}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {

	//获取网络端点中的veth的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("Fail config endpoint : %v", err)
	}

	/*
		将容器的网络端点加入到容器的网络空间中,并使这个函数下面的操作都在这个网络空间中进行
		需要对defer func()()的运行机制有着深入的理解,
		1. 当要进入容器的Net Namespace,只需要调用defer enterContainerNetns(&peerLink, cinfo)(),则在当前
		函数体结束之前都会在容器的Net Namespace中.
		2. 在调用enterContainerNetns(&peerLink, cinfo)()时会使当前执行的函数进入容器的Net Namespace中,在这里就是configEndpointIpAddressAndRoute函数
		,而用了defer后,会在函数体结束时执行返回的恢复函数指针,并会在函数结束之后恢复到原来所在的网络空间.

	*/
	defer enterContainerNetns(&peerLink, cinfo)()

	/*
		获取容器的IP和网段,用于配置容器内部接口地址
		比如容器IP是192.168.1.2 ,网络的网段是129.168.1.0/24
		那么这里的interfaceIP字符串就是192.168.1.2/24 ,用于容器内Veth端点配置
	*/
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	if err = setInteraceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("设置 Endpoint IP Error %v,%s", ep.Network, err)
	}

	//启动容器内的Veth
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	//Net Namespace中默认本地地址127.0.0.1的"lo"网卡是关闭的,需要启动它保证容器访问自己的请求
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	//设置访问容器内的外部请求都通过容器内的Veth端点访问
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	//构建要添加的路由数据,包括网络设备,网关IP以及目的网段
	//相当于route add -net 0.0.0.0/0 gw {Bridge网桥地址} dev {容器内的Veth端点设备}
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}

	//调用routeadd,添加路由到容器的网络空间
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

/*
	enterContainerNetns将容器的网络端点加入到容器的网络空间中
	并锁定当前程序所执行的线程,使当前线程进入到容器的网络空间
	返回值是个函数指针,执行这个返回函数才会退出容器的网络空间,回归到宿主机的网络空间
*/
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {

	//找到容器所在的Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("Error get container net namespace .%v", err)
	}

	//获得文件的文件描述符
	nsFD := f.Fd()

	//锁定当前程序所执行的线程,不锁定的话,Go语言的goroutine可能会被调度到其他的线程上去.无法保证一直所需的网络空间中
	runtime.LockOSThread()

	//Similar to: `ip link set $link netns $ns`
	//将veth移动到容器的Net namespace中来
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("Error set link netns %v", err)
	}

	//获得当前网络的Net Namespace,以便后面从容器的Net Namespace中退出回到原本网络的Namespace中
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("Error get current netns ,%v", err)
	}

	//调用Set方法,将当前进程加入到容器的Net Namespace中
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("Error set netns ,%v", err)
	}

	//返回之前Net Namespace的函数,在容器的网络空间中,完成了容器配置之后就使用此函数将程序回复到原来的Namespace中
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}

}

//configPortMapping 配置端口映射
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {

	for _, pm := range ep.PortMapping {
		//分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port Mapping format error %v", pm)
			continue
		}
		//将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])

		//执行iptables命令,添加端口映射转发规则
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output :%v", output)
			continue
		}
	}
	return nil
}

package network

import (
	"encoding/json"
	"net"
	"os"
	"path"
	"strings"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.json"

type IPAM struct {
	//分配文件存放位置
	SubnetAllocatorPath string
	//网段和位图算法的数组map,key是网段,value是分配的位图数组
	Subnets *map[string]string
}

//初始化一个IPAM对象,默认使用"/var/run/mydocker/network/ipam/subnet.json"作为分配信息存储位置
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网段地址分配信息
func (ipam *IPAM) load() error {
	//检查文件状态,若不存在,说明之前没有分配过,不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	//打开文件读取存储文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	subnetJson := make([]byte, 2000)

	//将subnetConfigFile中json类型的信息的长度存在n中,供接下来的反序列化使用
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	//将json类型的文件反序列化出IP的分配信息
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump allocation info %v", err)
		return err
	}
	return nil
}

// 存储网段地址分配信息
func (ipam *IPAM) dump() error {

	//检查存储文件所在文件夹是否存在,若不存在则创建,path.Split可以分隔目录和文件
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath) //接收的是目录
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}

	//打开存储文件os.O_TRUNC存在则清空,os.O_CREATE不存在则创建,os.O_WRONLY只写
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	//序列化ipam对象到json穿
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	// 将json信息写入文件
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}
	return nil

}

//Allocate 在网段中分配一个可用的ip地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {

	//存放网段地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	//从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("Error IPAM load allocation info %v ", err)
	}

	//net.IPNet.Mash.Size() 返回网段子网掩码的总长度和网段前面的固定位的长度

	one, size := subnet.Mask.Size()

	/*
			如果之前没有分配过这个网段,则初始化网段的分配配置
			ipam.Subnets存储的数据结构如下,用网段192.168.1.0/24表示
		 	ipam.Subnets:map[192.168.1.0][ 192.168.1.1,192.168.1.2,192.168.1.3,192.168.1.4,...].
			这个map的key是网段,value是在这个网段下具体的IP地址
	*/
	if _, exit := (*ipam.Subnets)[subnet.String()]; !exit {
		/*
			用"0"填满这个网段的配置,1<<uint8(size-one)表示这个网段中有多少可用的地址
			size-one 是子网掩码后面的网络位数,2^(size-one)表示网段中可用的IP数
			2^(size-one) 等价与 1<<uint8(size-one)
		*/
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	//遍历网段的位图数组
	for c := range (*ipam.Subnets)[subnet.String()] {
		//找到数组中为"0"的项和数组序号,即可以分配这个IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			//设置这个'0'的序号值为1,表示分配了这个IP
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			//Go中字符串创建后就无法修改,所以通过先转化成byte数组,修改后在转化成字符串赋值
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)

			//这个IP为初始IP,比如对于网段192.168.0.0/16来说,这里就是192.168.0.0
			ip = subnet.IP

			/*
				通过网段的IP和上面的偏移相加计算出分配的IP地址,由于IP地址是uint的一个数组,
				需要通过数组中的每一项加所需的值,比如网段是172.16.0.0/12,数组序号是65555,那么在[172.16.0.0]上
				依次加 [uint(65555 >> 24 )],[uint(65555 >> 16 )],[uint(65555 >> 8 )],[uint(65555 >> 0 )]
				即[0,1,0,19].那么获得的IP就是172.17.0.19
			*/
			for t := uint(4); t > 0; t-- {
				[]byte(ip)[4-t] += uint8(c >> (t - 1) * 8)
			}
			ip[3]++
			break
		}
	}

	// 调用dump将分配的结果保存到文件中
	ipam.dump()
	return

}

// Release 释放IP地址
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 从文件中加载网段的分配信息
	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump IPAM load info %v", err)
	}

	//计算IP地址在网段位图数组中的索引位置
	c := 0

	//将IP地址转换成4个字节的表示方式
	releaseIP := ipaddr.To4()
	//由于IP地址是从1开始分配的,所有换成索引是需要减一
	releaseIP[3] -= 1

	for t := uint(4); t > 0; t-- {
		//与分配IP相反,释放IP获得索引的方式是IP地址的每一位相减之后分别左移,将对应的数值加到索引上
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	//将分配的位图数组中索引位置的值赋值为0
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	//保存释放掉IP之后的网段IP分配信息
	ipam.dump()

	return nil

}

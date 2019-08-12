package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/network"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups/subsystems"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
)

//Run 容器启动入口
func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, volume string,
	containerName string, imageName string, envSlice []string, nw string, portmapping []string) {

	// 使用生成的随机数作为container ID
	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}

	/*
		这里的 Start 方法是真正开始前面创建好的 command 的调用，它首先会 clone 出来一个 name space 隔离的 进程，然后在子进程中，
		调用／ proc/self/exe ，也就是调用自己，发送 in it 参数，调用我们写的Init 方法， 去初始化容器的一些资源。

		父进程与子进程之间使用pipe进行通信
	*/
	parent, wirtePipe := container.NewParentProcess(tty, volume, containerName, imageName, envSlice)

	if parent == nil {
		log.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	//记录容器信息
	containerName, err := recodeContainerInfo(parent.Process.Pid, comArray, containerName, containerID, volume)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return
	}

	//use mydocker-cgroup as cgroup name
	//创建cgroup manager，并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager(containerID)

	//设置容器资源限制
	//fmt.Println("初始化Cgroup")
	cgroupManager.Set(res)
	//将容器进程加入到各个Subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)
	defer cgroupManager.Destory()

	if nw != "" {
		network.Init()
		containerInfo := &container.ContainerInfo{
			Id:          containerID,
			Pid:         strconv.Itoa(parent.Process.Pid),
			Name:        containerName,
			PortMapping: portmapping,
		}
		log.Infof("nw info %s", nw)
		if err := network.Connect(nw, containerInfo); err != nil {
			log.Errorf("Error Connect NetWork %v", err) //BUG
			return
		}
	}
	// 将参数发送到容器中
	sendInitCommand(comArray, wirtePipe)

	if tty {
		//parent.Wait()原来用于父进程等待子进程,在交互式的容器中没有问题.
		//如果使用detach创建容器,就无需等待,创建容器之后,父进程就会退出.
		//这是init进程接管这个子进程
		parent.Wait()
		deleteContainerInfo(containerName)
		container.DeleteWorkSpace(volume, containerName)
	}

}

//recodeContainerInfo 记录容器信息
func recodeContainerInfo(containerPID int, commandArray []string, containerName string, id, volume string) (string, error) {

	//生成容器的10位数字ID
	createTime := time.Now().Format("2006/1/2 15:04:05")
	command := strings.Join(commandArray, "")
	//没有制定容器名时一ID替代
	if containerName == "" {
		containerName = id
	}

	//生成容器信息结构体的实例
	containerInfo := &container.ContainerInfo{
		Id:         id,
		Pid:        strconv.Itoa(containerPID),
		Command:    command,
		CreateTime: createTime,
		Status:     container.RUNNING,
		Name:       containerName,
		Volume:     volume,
	}

	//将容器信息的对象json序列化成为字符串
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Recode contaienr info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	//完整的存储容器信息的路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirURL, 0622); err != nil {
		log.Errorf("Mkdir error %s error %v", dirURL, err)
		return "", err
	}

	//创建最终的配置文件--config.json文件
	fileName := dirURL + "/" + container.ConfigName
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return "", err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return "", err
	}
	return containerName, nil

}

//删除容器信息
func deleteContainerInfo(containerID string) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerID)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

//sendInitCommand 将信息发送给子进程
func sendInitCommand(comArray []string, wirtePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	wirtePipe.WriteString(command)
	wirtePipe.Close()
}

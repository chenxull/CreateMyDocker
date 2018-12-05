package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

// 系统调用kill发送信号给进程,去杀死主进程
/*
	1. 获取容器PID
	2. 对该PID发送kill信号
	3. 修改容器信息
	4. 将修改过的信息重新写入容器文件中
*/
func stopContainer(containerName string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("Get container pid by name %s error %v", containerName, err)
		return
	}

	//将string类型的pid转化为int型
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		log.Errorf("Conver pid from string to int error %v", err)
		return
	}

	//调用kill杀死进程
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		log.Errorf("Stop container %s error %v", containerName, err)
		return
	}

	//修改容器状态
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("Get containerinfo error %v", err)
		return
	}

	containerInfo.Status = container.STOP
	containerInfo.Pid = " "

	//将修改过后的信息序列化成json字符串
	newcontentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Json marshal %s error %v", containerName, err)
		return
	}

	//将修改后的内容写入对应目录中
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName

	if err := ioutil.WriteFile(configFilePath, newcontentBytes, 0622); err != nil {
		log.Errorf("WriteFile %s error %v", configFilePath, err)
		return
	}
	fmt.Println(containerName, "成功停止运行")

}

//因为使用的是Mydocker stop 容器名 的方式,所以需要使用容器名获得容器信息
func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName

	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Errorf("ReadFile %s error %v", configFilePath, err)
		return nil, err
	}

	var containerInfo container.ContainerInfo

	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		log.Errorf(" GetContainerInfoByName Unmarshal error %v ", err)
		return nil, err
	}
	return &containerInfo, nil

}

func removeContainer(containerName string) {
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("Get container %s info error %v", containerName, err)
		return
	}

	//只删除STOP状态的容器
	if containerInfo.Status != container.STOP {
		log.Errorf("Couldn`t remove running container %v", err)
		return
	}

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove file %s error %v", dirURL, err)
		return
	}
	container.DeleteWorkSpace(containerInfo.Volume, containerName)
}

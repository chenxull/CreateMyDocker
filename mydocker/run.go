package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups/subsystems"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {

	fmt.Println("ready to create NewParentProcess\n")
	parent, wirtePipe := container.NewParentProcess(tty)
	fmt.Println("NewParentProcess is Created\n")
	if parent == nil {
		log.Error("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	//use mydocker-cgroup as cgroup name
	//创建cgroup manager，并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup2")

	//设置容器资源限制
	fmt.Println("Run::init Cgrouplimit ")
	cgroupManager.Set(res)
	//将容器进程加入到各个Subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)
	defer cgroupManager.Destory()
	sendInitCommand(comArray, wirtePipe)
	parent.Wait()
	os.Exit(-1)
}

//sendInitCommand 将信息发送给子进程
func sendInitCommand(comArray []string, wirtePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("SendPipe::command all is %s", command)
	wirtePipe.WriteString(command)
	wirtePipe.Close()
}

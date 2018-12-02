package main

import (
	"os"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups/subsystems"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {
	parent, wirtePipe := container.NewParentProcess(tty)

	if parent == nil {
		log.Error("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	//use mydocker-cgroup as cgroup name
	//创建cgroup manager，并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	//设置容器资源限制
	cgroupManager.Set(res)
	//将容器进程加入到各个Subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)

	sendInitCommand(comArray, wirtePipe)
	parent.Wait()
	os.Exit(-1)
}

func sendInitCommand(comArray []string, wirtePipe *os.File) {

}

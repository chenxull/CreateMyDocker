package cgroups

import (
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups/subsystems"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

// CgroupManager 把不同Subsystem中的cgroup管理起来，并与容器建立关系
type CgroupManager struct {
	//cgroup在hierarchy中的路径，相当于创建的cgroup目录相对于各个root cgroup目录的路径
	Path string

	//资源配置
	Resource *subsystems.ResourceConfig
}

//NewCgroupManager 创建cgroup管理器
func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

//Apply 将PID加入到每个cgroup中
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Apply(c.Path, pid)
	}
	return nil
}

//Set 设置各个Subsystem挂载中的cgroup资源限制
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Set(c.Path, res)
	}
	return nil
}

//Destory 释放各个subsystem挂载中的cgroup
func (c *CgroupManager) Destory() error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			logrus.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}

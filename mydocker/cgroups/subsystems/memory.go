package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

//MemorySubSystem is memory subsystem的实现
type MemorySubSystem struct {
}

//Set is the interface method
//GetCgroupPath 的作用是获取当前 subsystem 在虚拟文件系统中的路径,具体来说就是找到对应Subsystem挂载的hierarchy相对路径对应的cgroup在虚拟文件系统中的路径
func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
		//fmt.Printf("subsysCgropuPath :%s\n", subsysCgroupPath)
		if res.MemoryLimit != "" {
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup memory fail %v", err)
			}
			//fmt.Println("Set memory limit info  into memory.limit_in_bytes!\n ")
		}
		return nil
	} else {
		return nil
	}
}

//Remove is the interface method
func (s *MemorySubSystem) Remove(cgroupPath string) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		return os.Remove(subsysCgroupPath)
	} else {
		return nil
	}
}

//Apply is the interface method
func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		//fmt.Println("apply memory limit successful!\n ")
		return nil
	} else {
		return nil
	}
}

//Name is the interface method
func (s *MemorySubSystem) Name() string {
	return "memory"
}

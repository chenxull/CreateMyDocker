package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

//FindCgroupMountpoint used to find out the CgroupMount point
func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// 根据cgroup的类型,查找起挂载路径
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {

			if opt == subsystem {
				//fmt.Printf("OPT::%s\n", opt)
				return fields[4] //返回正确的路径

			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return ""
}

//GetCgroupPath 得到cgroup在文件系统中的绝对路径
// GetCgroupPath 函数是找到对应 subsystem 挂载 的 hierarchy 相对路径对应的 cgroup 在虚拟文件 系统中的路径 ，
// 然后通过这个目录 的读写去操作 cgroup 。
func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsystem)
	//fmt.Printf("GetCgroupPath:: %s\n", cgroupRoot)

	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {
			} else {
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		return path.Join(cgroupRoot, cgroupPath), nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}

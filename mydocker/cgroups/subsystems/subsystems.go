package subsystems

//ResourceConfig used to restrict MemoryLimit,CPUShare,cpuSet
// 实现了三种资源的限制
type ResourceConfig struct {
	MemoryLimit string
	CpuShare    string
	CpuSet      string
}

//Subsystem 接口，每个接口有下面四个实现
type Subsystem interface {
	//返回Subsystem的名字
	Name() string
	//设置某个cgroup在这个Subsystem中的资源限制
	Set(path string, res *ResourceConfig) error
	//将进程添加进cgroup
	Apply(path string, pid int) error
	//移除某个cgroup
	Remove(path string) error
}

//通过不同的Subsystem初始化实例 创建资源限制处理链数组
var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)

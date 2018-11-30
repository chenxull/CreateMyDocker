package container

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

/*
这里是父进程，也就是当前进程执行的内容，根据上一章介绍的内容，应该比较容易明白。
 1. 这里的／proc/self/exe 调用中，／ proc/self ／指的是当前运行进程自己的环境， 调用了自己，使用这种方式对创建出来的进程进行初始化
 2 . 后面的 args 是参数，其中 init 是传递给本进程的第一个参数，在本例中，其实就是会去调用 initCornmand 去初始化进程的一些环境和资源
 3. 下面的 clone 参数就是去 fork 出来一个新进程，并且使用了 name space 隔离新创建的进程和外部环境。
 4. 如果用户指定了－ ti 参数，就需要把当前进程的输入输出导入到标准输入输出上
*/

func NewParentProcess(tty bool, command string) *exec.Cmd {
	args := []string{"init", command}
	cmd := exec.Command("/proc/self/exe", args...)
	logrus.Infof("DEBUG:: %s", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}

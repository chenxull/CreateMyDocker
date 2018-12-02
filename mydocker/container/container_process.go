package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

/*
这里是父进程，也就是当前进程执行的内容，根据上一章介绍的内容，应该比较容易明白。
 1. 这里的／proc/self/exe 调用中，／ proc/self ／指的是当前运行进程自己的环境， 调用了自己，使用这种方式对创建出来的进程进行初始化
 2 . 后面的 args 是参数，其中 init 是传递给本进程的第一个参数，在本例中，其实就是会去调用 initCornmand 去初始化进程的一些环境和资源
 3. 下面的 clone 参数就是去 fork 出来一个新进程，并且使用了 name space 隔离新创建的进程和外部环境。
 4. 如果用户指定了－ ti 参数，就需要把当前进程的输入输出导入到标准输入输出上

 上述这种调用方法存在bug，当用户输入的参数很长，或则带有一些特殊字符，上述方案就会失败。runC实现的方案
 是通过匿名管道来实现父子进程之间通信的.
*/

//NewParentProcess 父进程
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {

	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New Pipe error %v", err)
		return nil, nil
	}
	fmt.Println("Creating ParentProcess NO.1\n ")
	//cmd := exec.Command("/proc/self/exe", "init") // 怎么通过这个init去调用initCommand？/proc/se;f/exe就是调用自己，发送init参数，调用initcommand
	initCmd, err := os.Readlink("/proc/self/exe")
	if err != nil {
		log.Errorf("get init process error %v", err)
	}
	cmd := exec.Command(initCmd, "init")
	fmt.Println("Creating ParentProcess NO.2\n ")

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}

//NewPipe 创建父子进程间的通信管道
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, err
}

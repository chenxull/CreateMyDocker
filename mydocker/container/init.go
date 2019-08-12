package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

/*
	RunContainerInitProcess读取父进程传递过来的参数后，在子进程内进行了执行
	这里的 init 函数是在容器内部执行的，也就是说， 代码执行到这里后 ， 容器所在的进程其实就已经创建出来了，
	这是本容器执行的第一个进程。 使用 mount 先去挂载 proc 文件系统，以便后面通过 ps 等系统命令去查看当前进程资源的情况。
*/
func RunContainerInitProcess() error {

	// 读取父进程传递过来的参数,从Run中发送过来
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user command error ,cmdArray is nil")
	}

	// 更换root文件系统,将原有的文件系统给替换掉,使用镜像中的文件系统
	setUpMount()

	//argv := []string{command}
	//调用LookPath可以在系统的PATH里面寻找命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		log.Errorf("Exec loop path error %v", err)
		return err
	}
	log.Infof("Find path %s", path)
	// 在当前目录下执行

	/*
		正是这个系统调用实现 了完成初 始化动作井将用户进程运行起来 的操作。
		容器创建之后 ， 执行的第一个进程并不是用户的进程 ， 而是 init 初始化的进程。 这时候 ，如果通过 p s 命令查 看就会发现 ，
		 容器内第一个进程变成了自己 的 init， 这和预想 的是不一样的 。你可能会想 ， 大 不了 把第 一个进程给 kill 掉。
		但这里又有一个令人头疼 的问 题 ， PID 为 1 的进程是不 能被 kill 掉 的，如果该进程被 kill 掉 ， 我们 的容器也就退 出 了。


		syscall.Exec 这个方法 ， 其实最终调用了 Kernel 的 int execve( const char 咱lename, char *const argv[], cha r *const envp［］）；这个系统函数。
		它 的作用 是执行 当前 filename 对应的程序。它 会覆盖当前进程的镜像、数据和堆械等信息，包括 PID ， 这些都会被将要运行的进程覆盖掉。

		将用户指定的进程运行起来，把最初的 init 进程给替换掉，这样当 进入到容器内部的时候，就会发现容器内的第一个程序就是我们指定的进程了。
		这其实也是目 前 Docker 使用的容器引擎 rune 的实现方式之一。
	*/
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Error(err.Error())
	}
	return nil
}

//获得父进程传来过来的管道的信息
func readUserCommand() []string {
	//uintptr指的就是index为3的文件描述符，也就是传递过来的管道的一端
	pipe := os.NewFile(uintptr(3), "pipe")
	defer pipe.Close()
	msg, err := ioutil.ReadAll(pipe)
	log.Infof("Pipe::recive pipe Command\n ")
	// fmt.Println("Pipe::recive pipe Command\n ")
	if err != nil {
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func pivotRoot(root string) error {
	/*
		为了使当前root的老root和新root在不同的文件系统下，我们把root重新mount一次。 bind mount是把相同的内容换了一个挂载点的挂载方法。
	*/
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error :%v", err)
	}
	//创建rootfs/.pivot_root存储old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	//使用pivot_root到新的rootfs，老的rootfs现在挂载到了rootfs/.pivot_root中，然后使new_root成为新的root文件系统 root-->pivoDir
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}
	//修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("Chdir / %v", err)
	}
	pivotDir = filepath.Join("/", ".pivot_root")

	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("umount pivot_root dir %v", err)
	}
	//删除临时文件
	return os.Remove(pivotDir)

}

// init挂载点

func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Get current location error %v", err)
		return
	}
	log.Infof("Get current location is %s", pwd)
	//将当前目录挂载成为新的rootfs
	pivotRoot(pwd)

	//moune proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755s")
}

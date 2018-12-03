package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func NewParentProcess(tty bool, volume string) (*exec.Cmd, *os.File) {

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
	mntURL := "/root/mnt/"
	rootURL := "/root/"
	NewWorkSpace(rootURL, mntURL, volume)
	//rootfs的挂载目录
	cmd.Dir = mntURL
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

//NewWorkSpace 用来创建容器的文件系统
func NewWorkSpace(rootURL string, mntURL string, volume string) {
	CreateReadOnlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)
	//判断volume参数是否为空,当volumeUrlExtract函数返回的字符数组长度为2，并且数据元素不为空时，则执行MountVolume挂载数据卷
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(rootURL, mntURL, volumeURLs)
			fmt.Println("挂载外部volume成功")
			log.Infof("%q", volumeURLs)
		}
	}
}

//volumeUrlExtract 解析volume字符串
func volumeUrlExtract(volume string) []string {
	var volumeURLs []string
	volumeURLs = strings.Split(volume, ":")
	return volumeURLs
}

//MountVolume 挂载数据卷
func MountVolume(rootURL string, mntURL string, volumeURLs []string) {

	//创建宿主机文件目录
	parentUrl := volumeURLs[0]
	if err := os.MkdirAll(parentUrl, 0777); err != nil {
		fmt.Printf("host volume创建失败 %s", parentUrl)
		log.Infof("Mkdir parent dir %s error. %v", parentUrl, err)
	}
	fmt.Println("外部volume创建成功")

	//在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	containerVolumeURL := mntURL + containerUrl
	if err := os.MkdirAll(containerVolumeURL, 0777); err != nil {
		fmt.Println("容器数据卷创建失败")
		log.Infof("Mkdir container dir %s error. %v", containerVolumeURL, err)
	}

	//把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Mount volum failed. %v", err)
	}

}

// CreateReadOnlyLayer 将busybox.tar解压到busybox目录下，作为容器的只读层readonlylayer
func CreateReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	exist, err := PathExists(busyboxURL)

	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v ", busyboxURL, err)
	}
	//创建busyboxURL文件夹并将tar文件解压到此busybox中，作为ReadOnlyLayer
	if exist == false {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("unTar dir %s error %v", busyboxTarURL, err)
		}
	}
}

// CreateWriteLayer 创建一个名为writeLayer的文件夹作为容器唯一的可写层writeLayer
func CreateWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.Mkdir(writeURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error .%v", writeURL, err)
	}
}

// CreateMountPoint 使用mnt文件夹作为ReadOnlyLayer和WriteLayer的挂载点
func CreateMountPoint(rootURL string, mntURL string) {
	//创建mnt文件夹，作为挂载点
	if err := os.Mkdir(mntURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error.%v", mntURL, err)
	}

	//需要挂载的文件夹,将writeLayer写在前面，此时的一些读写操作都是在writelayer上进行的
	dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("挂载文件失败")
		log.Errorf("%v", err)
	}
}

//DeleteWorkSpace 当容器退出时，删除aufs文件系统
func DeleteWorkSpace(rootURL string, mntURL string, volume string) {

	if volume != "" {
		volumeURLS := volumeUrlExtract(volume)
		length := len(volumeURLS)
		if length == 2 && volumeURLS[0] != "" && volumeURLS[1] != "" {
			DeleteMountPointWithVolume(rootURL, mntURL, volumeURLS)
		} else {
			DeleteMountPoint(rootURL, mntURL)
		}
	} else {
		DeleteMountPoint(rootURL, mntURL)
	}

	DeleteWriteLayer(rootURL)
}

//DeleteMountPoint 结束文件的挂载
func DeleteMountPoint(rootURL string, mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("umount失败")
		log.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		fmt.Println("删除mntURL失败")
		log.Errorf("Remove dir %s error %v", mntURL, err)
	}
}

//DeleteWriteLayer 删除writeLayer文件夹
func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		fmt.Println("删除writeLayer文件夹失败")
		log.Errorf("Remove dir %s error %v", writeURL, err)
	}
}

func DeleteMountPointWithVolume(rootURL string, mntURL string, volumeURLs []string) {

	//Umount 容器内的volume挂载
	containerUrl := mntURL + volumeURLs[1]
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount volume failed %v", err)
	}

	//Umount 整个容器的文件的系统
	cmd = exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount mountpoint failed %v", err)
	}

	//删除容器挂载文件夹
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("Remove  dir %s failed.%v ", mntURL, err)
	}
}

//PathExists 判断文件路径是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, nil
}

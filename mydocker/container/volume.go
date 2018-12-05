package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

//NewWorkSpace 用来创建容器的文件系统
func NewWorkSpace(volume, imageName, containerName string) {
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	CreateMountPoint(containerName, imageName)
	//判断volume参数是否为空,当volumeURLExtract函数返回的字符数组长度为2，并且数据元素不为空时，则执行MountVolume挂载数据卷
	if volume != "" {
		volumeURLs := volumeURLExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(volumeURLs, containerName) //bug:: 将containerName写成imageName
			log.Infof("NewWorkSpace volume urls %q", volumeURLs)
		} else {
			log.Infof("Volume parameter input is not correct")
		}
	}
}

//volumeURLExtract 解析volume字符串
func volumeURLExtract(volume string) []string {
	var volumeURLs []string
	volumeURLs = strings.Split(volume, ":")
	return volumeURLs
}

//MountVolume 挂载数据卷,根据用户输入的volume参数获取相应要挂载的宿主机数据卷URL和容器中的挂载点URL,并挂载数据卷.
// 容器的挂载点为MntUrl + containerName + containerURL 命名
func MountVolume(volumeURLs []string, containerName string) error {

	//创建宿主机文件目录
	parentURL := volumeURLs[0]
	//MkdirAll是建立多级目录
	if err := os.Mkdir(parentURL, 0777); err != nil {
		log.Infof("Mkdir parent dir %s error. %v", parentURL, err)
	}

	//在容器文件系统里创建挂载点
	containerURL := volumeURLs[1]
	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerVolumeURL := mntURL + "/" + containerURL
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
		fmt.Println("容器数据卷创建失败")
		log.Infof("Mkdir containerVolume dir %s error. %v", containerVolumeURL, err)
	}

	//把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentURL
	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL).CombinedOutput()
	if err != nil {
		log.Errorf("Mount volume failed %v", err)
		return err
	}

	return nil

}

// CreateReadOnlyLayer 将busybox.tar解压到busybox目录下，作为容器的只读层readonlylayer
//根据用户输入的镜像为每个容器创建只读层,镜像解压出来的只读层以RootUrl + imageName 命名
func CreateReadOnlyLayer(imageName string) error {

	unTarFolderURL := RootUrl + "/" + imageName + "/"
	imageURL := RootUrl + "/" + imageName + ".tar"

	exist, err := PathExists(unTarFolderURL)

	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v ", unTarFolderURL, err)
		return err
	}
	//创建busyboxURL文件夹并将tar文件解压到此busybox中，作为ReadOnlyLayer
	if !exist {
		if err := os.MkdirAll(unTarFolderURL, 0622); err != nil {
			log.Errorf("Mkdir dir %s error. %v", unTarFolderURL, err)
		}

		if _, err := exec.Command("tar", "-xvf", imageURL, "-C", unTarFolderURL).CombinedOutput(); err != nil {
			log.Errorf("UnTar dir %s error %v", unTarFolderURL, err)
		}
	}
	return nil
}

// CreateWriteLayer 创建一个名为writeLayer的文件夹作为容器唯一的可写层writeLayer
// CreateWriteLayer 在路径WriteLayer + containerName 创建容器的读写层
func CreateWriteLayer(containerName string) {

	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.MkdirAll(writeURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error .%v", writeURL, err)
	}
}

// CreateMountPoint 创建容器的根目录,把容器的ReadOnlyLayer和WriteLayer层挂载到容器的根目录,成为容器的文件系统
func CreateMountPoint(containerName, imageName string) error {
	//创建mnt文件夹，作为挂载点
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if err := os.MkdirAll(mntUrl, 0777); err != nil {
		log.Errorf("Mkdir mountPoint dir %s error.%v", mntUrl, err)
		return err
	}

	//需要挂载的文件夹,将writeLayer写在前面，此时的一些读写操作都是在writelayer上进行的
	tmpWriteLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	tmpImageLocation := RootUrl + "/" + imageName
	mntURL := fmt.Sprintf(MntUrl, containerName)
	dirs := "dirs=" + tmpWriteLayer + ":" + tmpImageLocation

	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL).CombinedOutput()

	if err != nil {
		log.Errorf("Run command for creating mount point failed %v", mntURL)
		return err
	}
	return nil
}

//DeleteWorkSpace 当容器退出时，删除aufs文件系统
func DeleteWorkSpace(volume, containerName string) {

	if volume != "" {
		volumeURLs := volumeURLExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteMountPointWithVolume(volumeURLs, containerName)
		} else {
			DeleteMountPoint(containerName)
		}
	} else {
		DeleteMountPoint(containerName)
	}

	DeleteWriteLayer(containerName)
}

//DeleteMountPoint 结束文件的挂载
func DeleteMountPoint(containerName string) error {
	//寻找挂载点
	mntURL := fmt.Sprintf(MntUrl, containerName)
	_, err := exec.Command("umount", mntURL).CombinedOutput()

	if err != nil {
		log.Errorf("Umount mountpoint dir %s error %v", mntURL, err)
		return err
	}

	if err := os.RemoveAll(mntURL); err != nil {

		log.Errorf("Remove  mountpoint dir %s error %v", mntURL, err)
		return err
	}
	return nil
}

//DeleteWriteLayer 删除writeLayer文件夹
func DeleteWriteLayer(contaierName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, contaierName)
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("Remove WriteLayer dir %s error %v", writeURL, err)
	}
}

//DeleteMountPointWithVolume 删除挂载数据卷容器的文件系统,先umount容器对应挂载点,在umount容器的文件系统
func DeleteMountPointWithVolume(volumeURLs []string, containerName string) error {

	mntURL := fmt.Sprintf(MntUrl, containerName)
	//Umount 容器内的volume挂载
	containerURL := mntURL + "/" + volumeURLs[1]

	if _, err := exec.Command("umount", containerURL).CombinedOutput(); err != nil {
		log.Errorf("Umount volume %s failed %v", containerURL, err)
		return err
	}

	if _, err := exec.Command("umount", mntURL).CombinedOutput(); err != nil {
		log.Errorf("Umount mountPoint %s failed %v", mntURL, err)
		return err
	}

	//删除容器挂载文件夹
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("Remove mountPoint dir %s failed.%v ", mntURL, err)
	}
	return nil
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

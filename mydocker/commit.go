package main

import (
	"fmt"
	"os/exec"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

// 制作{imageName}.tar镜像
func commitContainer(containerName, imageName string) {

	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	mntURL += "/"

	imageTar := container.RootUrl + "/" + imageName + ".tar"

	fmt.Printf("镜像打包成功")
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error . %v", mntURL, err)
	}
}

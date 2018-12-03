package main

import (
	"fmt"
	"os/exec"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

func commitContainer(imageName string) {
	mntURL := "/root/mnt"
	imageTar := "/root/" + imageName + ".tar"
	fmt.Printf("镜像打包成功")
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error . %v", mntURL, err)
	}
}

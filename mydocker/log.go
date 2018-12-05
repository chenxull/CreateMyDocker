package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
)

// logContainer 打印日志信息
func logContainer(containerName string) {

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := dirURL + container.ContainerLogFile

	//打开日志文件
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		log.Errorf("Log contianer open file %s error %v", logFileLocation, err)
		return
	}

	//读取日志文件内容
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("Log container read file %s error %v", file, err)
		return
	}

	//打印日志
	fmt.Fprint(os.Stdout, string(content))

}

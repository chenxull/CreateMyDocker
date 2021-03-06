package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
	_ "github.com/chenxull/mydocker/CreateMyDocker/mydocker/nsenter"
)

//ENV_EXEC_PID C语言中使用的环境变量
const ENV_EXEC_PID = "mydocker_pid"
const ENV_EXEC_CMD = "mydocker_cmd"

// ExecContainer 是指令exec的实现
func ExecContainer(containerName string, comArray []string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("Exec container getContainerPidByName %s error %v", containerName, err)
		return
	}

	//把命令以空格为分隔符拼成一个字符串,便于传递
	cmdStr := strings.Join(comArray, " ")
	log.Infof("container Pid %s", pid)
	log.Infof("command %s", cmdStr)

	//下面使用参数exec,就是为了c代码再次执行一遍,这次执行是制定了环境变量
	// 再一次运行的时候已经指定了环境变量，所以 C 代码执行的时候就能拿到对应的环境变量 ， 便可以进入到指定的 Namespace 中进行操作了。
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	//获得环境变量
	containerEnvs := getEnvsByPid(pid)
	//将宿主机的环境变量和容器的环境变量都放到exec进程中
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		log.Errorf("Exec contaienr %s error %v", containerName, err)
	}

}

//根据容器的名字获得起PID
func getContainerPidByName(containerName string) (string, error) {

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName

	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}

	var containerInfo container.ContainerInfo
	//将文件内容反序列化成容器信息对象,然后然后对应的PID
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		log.Errorf("unmarshal contentBytes fail %v", err)
		return "", err
	}
	return containerInfo.Pid, nil
}

// 通过PID获取环境变量
func getEnvsByPid(pid string) []string {
	//进程的环境变量在/proc/PID/environ中
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Read environ file %s error %v", path, err)
		return nil
	}

	//多个环境变量中的分隔符是\u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}

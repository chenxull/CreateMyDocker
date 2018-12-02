package main

import (
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/cgroups/subsystems"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"

	"fmt"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/urfave/cli"
)

//这里定义了 runCommand 的 Flags ，其作用类似于运行命令时使用一来指定参数
var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit 
			mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
	},
	/* 这里是 run 命令执行的真正函数。
	1. 判断参数是否包含 command
	2. 获取用户指定的 command
	3. 调用 Run function 去准备启动容器
	*/
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		//cmd := context.Args().Get(0)

		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		tty := context.Bool("ti")

		resconfig := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuSet:      context.String("cpuset"),
			CpuShare:    context.String("cpushare"),
		}
		fmt.Println("runCommand is starting \n")
		Run(tty, cmdArray, resconfig)
		return nil
	},
}

//定义了InitCommand 的具体操作，此操作为内部方法，禁止外部调用
var initCommand = cli.Command{
	Name:  "init",
	Usage: " Init container process run user`s process in container .Do not call it outside",
	/*
		1. 获取传递过来的 command 参数
		2. 执行容器初始化操作
	*/
	Action: func(context *cli.Context) error {
		//log.Infof("init come on 1")
		cmd := context.Args().Get(0)
		log.Infof("InitCommand  %s", cmd)

		err := container.RunContainerInitProcess()
		return err
	},
}

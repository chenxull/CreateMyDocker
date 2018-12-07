package main

import (
	"os"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/network"

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
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
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
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringSliceFlag{
			Name:  " e",
			Usage: "set environment",
		},
		cli.StringFlag{
			Name:  "net",
			Usage: "contaienr network",
		},
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping",
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

		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
		createTty := context.Bool("ti")
		detach := context.Bool("d")
		envSlice := context.StringSlice("e")

		if createTty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}
		volume := context.String("v")
		resconfig := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuSet:      context.String("cpuset"),
			CpuShare:    context.String("cpushare"),
		}
		contaienrName := context.String("name")
		fmt.Println("runCommand is starting \n")

		Run(createTty, cmdArray, resconfig, volume, contaienrName, imageName, envSlice)
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

var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image,You need to add two param containerName and imageName",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 2 {

			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		commitContainer(containerName, imageName)
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Please input your container name")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {

		ListContainers()
		return nil
	},
}

//这个的执行过程是一个难点
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		//this is for callback,第二次进入本程序执行exec的时候,如果已经制定了环境变量,说明c代码已经执行,直接返回就行了,避免重复调用
		if os.Getenv(ENV_EXEC_PID) != "" {
			log.Infof("pid callback pid %s", os.Getgid())
			return nil
		}
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}

		containerName := context.Args().Get(0)

		var commandArray []string
		for _, arg := range context.Args().Tail() {
			commandArray = append(commandArray, arg)
		}

		ExecContainer(containerName, commandArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}

		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network command",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},

			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")

				}
				network.Init()
				err := network.CreateNetwork(context.String("driver"), context.String("subnet"), context.Args()[0])

				if err != nil {
					return fmt.Errorf("CreateNetwork Error : %v", err)
				}
				return
			},
		},
		{
			Name:  "list",
			Usage: " list container network",
			Action: func(context *cli.Context) error {
				network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove contianer network",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}

				network.Init()
				err := network.DeleteNetwork(context.Args()[0])
				if err != nil {
					return fmt.Errorf("remove network Error : %v", err)
				}
				return nil
			},
		},
	},
}

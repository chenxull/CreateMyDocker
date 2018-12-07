package main

import (
	"fmt"
	"os"

	log "github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/Sirupsen/logrus"
	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/github.com/urfave/cli"
)

const usage = `mydocker is a simple container runtime implementation.
	The purpose of this project is to learn how docker works and how to write a docker by ourselves,just for a 
	good job!.`

func main() {
	app := cli.NewApp()
	app.Name = "mydocker"
	app.Usage = usage
	fmt.Println("main::before Commands\n")
	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		commitCommand,
		listCommand,
		logCommand,
		execCommand,
		stopCommand,
		removeCommand,
		networkCommand,
	}

	// Init log
	app.Before = func(content *cli.Context) error {
		//Log as JSON instead of the defaul ASCII formatter.
		log.SetFormatter(&log.JSONFormatter{})

		log.SetOutput(os.Stdout)

		return nil
	}
	//fmt.Println("main::starting\n")
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

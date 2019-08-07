package network

import (
	"testing"

	"github.com/chenxull/mydocker/CreateMyDocker/mydocker/container"
)

func TestBridgeInit(t *testing.T) {
	d := BridgeNetWorkDriver{}
	_, err := d.Create("192.168.0.1/24", "chenxubridge")
	t.Logf("err : %v", err)
}

func TestBridgeConnect(t *testing.T) {
	ep := Endpoint{
		ID: "testcontainer",
	}

	n := Network{
		Name: "chenxubridge2",
	}
	d := BridgeNetWorkDriver{}
	err := d.Connect(&n, &ep)
	t.Logf("err : %v", err)
}

func TestNetworkConnect(t *testing.T) {
	cinfo := &container.ContainerInfo{
		Id:  "testcontainer",
		Pid: "15438",
	}

	d := BridgeNetWorkDriver{}
	n, err := d.Create("192.168.1.1/24", "chenxubridge3")
	t.Logf("err:%v", n)

	networks[n.Name] = n
	err = Connect(n.Name, cinfo)
	t.Logf("err : %v", err)
}

func TestLoad(t *testing.T) {
	n := Network{
		Name: "chenxubridge4",
	}
	n.load("/var/run/mydocker/network/network/chenxubridge4")
	t.Logf("network :%v", n)
}

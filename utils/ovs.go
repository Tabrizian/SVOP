/**
 * File              : ovs.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 04.04.2019
 * Last Modified Date: 10.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/Tabrizian/SVOP/models"
	"golang.org/x/crypto/ssh"
)

func InstallOVS(vm models.VM) {
	cmd := "dpkg -l | grep openvswitch"
	stdOut, _ := RunCommand(vm, cmd)
	if string(stdOut) == "" {
		log.Print("Open vSwitch is not installed, installing now...")
		cmd = "sudo apt-get update && sudo apt-get install -y openvswitch-switch"
		log.Print("Open vSwitch is not installed, installing now...")
		output, _ := RunCommand(vm, cmd)
		log.Println(output)
	}
}

func SetController(vm models.VM, ctrlEndpoint string) {
	ovsName := "br1"
	cmd := fmt.Sprintf("sudo ovs-vsctl set-controller %s tcp:%s", ovsName, ctrlEndpoint)
	output, _ := RunCommand(vm, cmd)
	log.Println(output)
}

func SetOverlayInterface(vm models.VM, hostOverlayIP string) {
	log.Printf("Setting overlay for %s with IP %s\n", vm.Name, hostOverlayIP)
	InstallOVS(vm)
	RunCommand(vm, "sudo ovs-vsctl --may-exist add-br br1")
	if hostOverlayIP != "" {
		RunCommand(vm, "sudo ovs-vsctl --may-exist add-port "+
			"br1 br1-internal -- set interface br1-internal type=internal")
		cmd := fmt.Sprintf("sudo ifconfig br1-internal %s/24 mtu 1450 up", hostOverlayIP)
		RunCommand(vm, cmd)
	}
}

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func RunCommand(vm models.VM, cmd string) ([]byte, []byte) {
	sshConfig := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			PublicKeyFile("/home/iman/.ssh/id_rsa"),
			ssh.Password("savi"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", vm.IP[0]+":22", sshConfig)
	for err != nil {
		log.Printf("Failed to dial: %s", err)
		connection, err = ssh.Dial("tcp", vm.IP[0]+":22", sshConfig)
		time.Sleep(5000 * time.Millisecond)
	}
	log.Println("Initiating SSH connection")

	session, err := connection.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	log.Println("Creating new session")

	sessionStdOut, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create session pipe: %s", err)
	}

	sessionStdErr, err := session.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create session pipe: %s", err)
	}

	err = session.Run(cmd)
	if err != nil {
		log.Printf("Command status code problematic: %s", err)
	}

	stdOutByte, err := ioutil.ReadAll(sessionStdOut)
	if err != nil {
		log.Fatalf("Failed to read all of the SSH output: %s", err)
	}

	stdErrByte, err := ioutil.ReadAll(sessionStdErr)
	if err != nil {
		log.Fatalf("Failed to read all of the SSH output: %s", err)
	}
	log.Println(string(stdOutByte))
	log.Println("Error: " + string(stdErrByte))

	return stdOutByte, stdErrByte
}

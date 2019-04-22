/**
 * File              : ovs.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 04.04.2019
 * Last Modified Date: 15.04.2019
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
		RunCommand(vm, cmd)
	}
	cmd = "dpkg -l | grep sshpass"
	stdOut, _ = RunCommand(vm, cmd)
	if string(stdOut) == "" {
		cmd = "sudo apt-get install -y sshpass"
		log.Print("sshpass is not installed, installing now...")
		RunCommand(vm, cmd)
	}

	cmd = "sudo sed -i 's/ExecStart=.*/ExecStart=\\/usr\\/bin\\/dockerd -H fd:\\/\\/ -H 0\\.0\\.0\\.0/g' /lib/systemd/system/docker.service"
	RunCommand(vm, cmd)

	cmd = "docker run --volume=/:/rootfs:ro --volume=/var/run:/var/run:ro --volume=/sys:/sys:ro   --volume=/var/lib/docker/:/var/lib/docker:ro   --volume=/dev/disk/:/dev/disk:ro   --publish=8080:8080   --detach=true   --name=cadvisor --restart always  google/cadvisor:latest"
	RunCommand(vm, cmd)

	cmd = "curl -LO https://github.com/prometheus/node_exporter/releases/download/v0.17.0/node_exporter-0.17.0.linux-amd64.tar.gz"
	RunCommand(vm, cmd)

	cmd = "tar xzf node_exporter-0.17.0.linux-amd64.tar.gz && cd node_exporter-0.17.0.linux-amd64 && sudo mv ./node_exporter /usr/local/bin/"
	RunCommand(vm, cmd)

	cmd = "sudo useradd -rs /bin/false node_exporter"
	RunCommand(vm, cmd)

	cmd = `
[Unit] 
Description=Node Exporter 
After=network.target 

[Service] 
User=node_exporter 
Group=node_exporter 
Type=simple 
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
`

	RunCommand(vm, "sudo bash -c \"echo '"+cmd+"' > /etc/systemd/system/node_exporter.service\"")
	RunCommand(vm, "sudo systemctl daemon-reload")
	RunCommand(vm, "sudo systemctl start node_exporter")
	RunCommand(vm, "sudo systemctl restart docker")
	RunCommand(vm, "sudo systemctl enable node_exporter")

}

func SetController(vm models.VM, ctrlEndpoint string) {
	ovsName := "br1"
	cmd := fmt.Sprintf("sudo ovs-vsctl set-controller %s tcp:%s", ovsName, ctrlEndpoint)
	output, _ := RunCommand(vm, cmd)
	log.Println(output)
}

func SetOverlayInterface(vm models.VM, hostOverlayIP string) {
	log.Printf("Setting overlay for %s with IP %s\n", vm.Name, hostOverlayIP)
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
	return RunCommandOverSSH(vm.IP[0], cmd)
}

func RunCommandOverSSH(ip string, cmd string) ([]byte, []byte) {
	log.Printf("Running command %s\n", cmd)
	sshConfig := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			PublicKeyFile("/home/iman/.ssh/id_rsa"),
			ssh.Password("savi"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", ip+":22", sshConfig)
	for err != nil {
		log.Printf("Failed to dial: %s", err)
		connection, err = ssh.Dial("tcp", ip+":22", sshConfig)
		time.Sleep(5000 * time.Millisecond)
	}
	log.Println("Initiating SSH connection to " + ip)

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
		// If grep doens't provide any output this will be printed
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

func RunCommandFromOverlay(vm models.VM, cmd string, overlayIP string) ([]byte, []byte) {
	cmdToBeExec := fmt.Sprintf("sshpass -p savi ssh -o StrictHostKeyChecking=no ubuntu@%s %s", vm.OverlayIp, cmd)
	return RunCommandOverSSH(overlayIP, cmdToBeExec)
}

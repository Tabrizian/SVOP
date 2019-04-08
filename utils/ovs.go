/**
 * File              : ovs.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 04.04.2019
 * Last Modified Date: 08.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package utils

import (
    "io/ioutil"
    "log"

    "golang.org/x/crypto/ssh"
    "github.com/Tabrizian/SVOP/models"
)

func InstallOVS (vm models.VM) {
    cmd := "dpkg -l | grep openvswitch"
    stdOut, _ := RunCommand(vm, cmd)
    if string(stdOut) == "" {
        log.Print("Open vSwitch is not installed, installing now...")
        cmd = "sudo apt-get update && sudo apt-get install -y openvswitch-switch"
        RunCommand(vm, cmd)
        log.Print("Open vSwitch is not installed, installing now...")
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
        Auth: []ssh.AuthMethod{PublicKeyFile("/home/iman/.ssh/id_rsa")},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    connection, err := ssh.Dial("tcp", vm.IP[0] + ":22", sshConfig)
    if err != nil {
        log.Fatalf("Failed to dial: %s", err)
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

    return stdOutByte, stdErrByte
}


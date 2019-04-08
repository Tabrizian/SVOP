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
    "io"
    "log"
    "os"

    "golang.org/x/crypto/ssh"
    "github.com/Tabrizian/SVOP/models"
)

func installOVS (vm models.VM) {
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

func CreateSSHClient(ip string, cmd string) {
    sshConfig := &ssh.ClientConfig{
        User: "ubuntu",
        Auth: []ssh.AuthMethod{PublicKeyFile("/home/iman/.ssh/id_rsa")},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    connection, err := ssh.Dial("tcp", ip, sshConfig)
    if err != nil {
        log.Fatalf("Failed to dial: %s", err)
    }
    log.Println("Initiating SSH connection")

    session, err := connection.NewSession()
    if err != nil {
        log.Fatalf("Failed to create session: %s", err)
    }
    log.Println("Creating new session")

    defer session.Close()

    sessionStdOut, err := session.StdoutPipe()
    if err != nil {
        log.Fatalf("Failed to create session pipe: %s", err)
    }

    go io.Copy(os.Stdout, sessionStdOut)

    err = session.Run(cmd)
    if err != nil {
        log.Fatalf("Failed to run the command: %s", err)
    }
    log.Println("Reached the end of function")
}

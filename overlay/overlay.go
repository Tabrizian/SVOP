/**
* File              : overlay.go
* Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
* Date              : 10.04.2019
* Last Modified Date: 22.04.2019
* Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package overlay

import (
	"fmt"
	"log"

	"math/rand"

	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/utils"
)

var vnis []int

type Host struct {
	Gateway   bool
	OverlayIP string
}

func generateVNI() int {
	vni := rand.Intn(6000)

	for {
		exist := false
		for item := range vnis {
			if item == vni {
				exist = true
				break
			}
		}
		if exist == false {
			vnis = append(vnis, vni)
			break
		}
		vni = rand.Intn(6000)
	}

	return vni
}

func connectNodes(node1 models.VM, node2 models.VM) {
	log.Printf("Making VXLAN between %s and %s\n", node1.Name, node2.Name)

	// Ensure OVS daemon is up and running in both nodes
	utils.InstallOVS(node1)
	utils.InstallOVS(node2)

	vni := generateVNI()
	cmd := fmt.Sprintf("sudo ovs-vsctl add-port br1 %s -- set interface %s type=vxlan options:remote_ip=%s options:key=%v", node1.Name+"-"+node2.Name, node1.Name+"-"+node2.Name, node2.IP[0], vni)
	utils.RunCommand(node1, cmd)
	cmd = fmt.Sprintf("sudo ovs-vsctl add-port br1 %s -- set interface %s type=vxlan options:remote_ip=%s options:key=%v", node2.Name+"-"+node1.Name, node2.Name+"-"+node1.Name, node1.IP[0], vni)
	utils.RunCommand(node2, cmd)
}

func fixConnectionFormat(connection interface{}) map[string]string {

	connectionAsserted := connection.(map[interface{}]interface{})
	connectionString := make(map[string]string)
	for key, value := range connectionAsserted {
		switch key := key.(type) {
		case string:
			switch value := value.(type) {
			case string:
				connectionString[key] = value
			}
		}
	}
	return connectionString
}

func ExtractSWs(overlay map[string]interface{}) map[string]struct{} {
	var switches map[string]struct{}

	switches = make(map[string]struct{})

	for sw, connections := range overlay {
		if _, ok := switches[sw]; !ok {
			switches[sw] = struct{}{}
		}

		connectionsAsserted := connections.([]interface{})
		for _, connection := range connectionsAsserted {
			connectionString := fixConnectionFormat(connection)
			endpoint := connectionString["endpoint"]
			// Is it a host
			if _, ok := connectionString["ip"]; !ok {
				if _, ok := switches[endpoint]; !ok {
					switches[endpoint] = struct{}{}
				}
			}
		}
	}

	return switches
}

func ExtractHosts(overlay map[string]interface{}) map[string]Host {
	var hosts map[string]Host

	hosts = make(map[string]Host)

	for _, connections := range overlay {
		connectionsAsserted := connections.([]interface{})
		for _, connection := range connectionsAsserted {
			connectionString := fixConnectionFormat(connection)
			endpoint := connectionString["endpoint"]
			// Is it a host
			if _, ok := connectionString["ip"]; ok {
				if _, ok := hosts[endpoint]; !ok {
					host := Host{}
					if _, ok := connectionString["default"]; ok {
						host = Host{
							OverlayIP: connectionString["ip"],
							Gateway:   true,
						}
					} else {
						host = Host{
							OverlayIP: connectionString["ip"],
							Gateway:   false,
						}
					}
				}
			}
		}
	}

	return hosts
}

func ConfigureSwitch(osClient utils.OpenStackClient, name string, vmConfiguration models.VMConfiguration) *models.VM {
	vmObj, err := models.CreateOrFindVM(&osClient, name, &vmConfiguration)
	log.Printf("Working on switch %s\n", vmObj.Name)
	if err != nil {
		log.Fatalf("VM creation failed")
	}
	vmObjD := *vmObj
	utils.SetOverlayInterface(vmObjD, "")
	utils.RunCommand(vmObjD, "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")
	return vmObj
}

func ConfigureHost(osClient utils.OpenStackClient, host Host, vmConfiguration models.VMConfiguration) *models.VM {
	vmObj, err := models.CreateOrFindVM(&osClient, host.Name, &vmConfiguration)
	log.Printf("Working on host %s\n", vmObj.Name)
	if err != nil {
		log.Fatalf("VM creation failed")
	}
	vmObjD := *vmObj
	utils.SetOverlayInterface(vmObjD, host.OverlayIP)
	utils.RunCommand(vmObjD, "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")
	if host.Gateway {
		utils.RunCommand(vmObjD, "sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE")
		utils.RunCommand(vmObjD, "sudo iptables -P FORWARD ACCEPT")
	}
	return vmObj
}

func DeployOverlay(osClient utils.OpenStackClient, overlay map[string]interface{}, vmConfiguration models.VMConfiguration, ctrlEndpoint string) {
	var switches map[string]models.VM
	var hosts map[string]models.VM
	var defaultOverlayAddress string
	var gatewayIp string

	switches = make(map[string]models.VM)
	hosts = make(map[string]models.VM)

	switchChannel := make(chan *models.VM)
	switchNames := ExtractSWs(overlay)

	hostChannel := make(chan *models.VM)
	hostNames := ExtractHosts(overlay)

	for sw, _ := range switchNames {
		go func(sw string) {
			log.Printf("Creating VM %s\n", sw)
			switchChannel <- ConfigureSwitch(osClient, sw, vmConfiguration)
		}(sw)
	}

	for host, _ := range hostNames {
		go func(host string) {
			log.Printf("Creating VM %s\n", host)
			hostChannel <- ConfigureHost(osClient, host, vmConfiguration)
		}(host)
	}

	for range hostNames {
		host := <-hostChannel
		hosts[host.Name] = *host
		defaultOverlayAddress = hostNames[host.Name].OverlayIP
		gatewayIp = host.IP[0]
	}

	for range switchNames {
		sw := <-switchChannel
		switches[sw.Name] = *sw
	}

	// for sw, connections := range overlay {
	// 	connectionsAsserted := connections.([]interface{})
	// 	vmSrc := switches[sw.Name]
	// 	for _, connection := range connectionsAsserted {
	// 		connectionString := fixConnectionFormat(connection)

	// 		endpoint := connectionString["endpoint"]
	// 		var endpointVM models.VM
	// 		_ = endpointVM

	// 		// Is it a host
	// 		if _, ok := connectionString["ip"]; ok {
	// 			if val, ok := hosts[endpoint]; ok {
	// 				endpointVM = val
	// 			} else {
	// 				vmObject, err := models.CreateOrFindVM(&osClient, endpoint, &vmConfiguration)
	// 				log.Printf("Working on VM %s\n", vmObject.Name)
	// 				if err != nil {
	// 					log.Fatalf("VM creation failed")
	// 				}
	// 				ip := connectionString["ip"]
	// 				vmObject.OverlayIp = ip
	// 				hosts[endpoint] = *vmObject
	// 				endpointVM = *vmObject
	// 				utils.SetOverlayInterface(hosts[endpoint], ip)
	// 				utils.RunCommand(hosts[endpoint], "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")

	// 				// Setup the NAT and configure iptables
	// 				if _, ok := connectionString["default"]; ok {
	// 					defaultOverlayAddress = ip
	// 					gatewayIp = vmObject.IP[0]
	// 					utils.RunCommand(endpointVM, "sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE")
	// 					utils.RunCommand(endpointVM, "sudo iptables -P FORWARD ACCEPT")
	// 				}
	// 			}
	// 		} else {
	// 			if val, ok := switches[endpoint]; ok {
	// 				endpointVM = val
	// 			} else {
	// 				vmObject, err := models.CreateOrFindVM(&osClient, endpoint, &vmConfiguration)
	// 				log.Printf("Working on VM %s\n", vmObject.Name)
	// 				if err != nil {
	// 					log.Fatalf("VM creation failed")
	// 				}
	// 				switches[endpoint] = *vmObject
	// 				endpointVM = *vmObject
	// 				utils.SetOverlayInterface(switches[endpoint], "")
	// 			}
	// 		}
	// 		connectNodes(vmSrc, endpointVM)
	// 	}
	// }

	log.Println("Replacing default gateway with the new gateway")
	for _, vm := range hosts {
		if vm.OverlayIp != defaultOverlayAddress {
			utils.RunCommand(vm, "sshpass -p savi ssh -o StrictHostKeyChecking=no ubuntu@"+gatewayIp+" bash -c 'until ping -c1 www.google.com >/dev/null 2>&1; do :; done'"+vm.OverlayIp)
			out, _ := utils.RunCommandFromOverlay(vm, "route | awk 'NR==3{print $2}'", gatewayIp)
			utils.RunCommandFromOverlay(vm, "sudo route add default gw "+defaultOverlayAddress, gatewayIp)
			utils.RunCommandFromOverlay(vm, "sudo route del default gw "+string(out), gatewayIp)
		}
	}

	log.Println("Setting the controller for switches")
	for _, sw := range switches {
		utils.RunCommand(sw, "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")
		utils.SetController(sw, ctrlEndpoint+":6633")
	}
}

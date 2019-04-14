/**
* File              : overlay.go
* Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
* Date              : 10.04.2019
* Last Modified Date: 14.04.2019
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

func DeployOverlay(osClient utils.OpenStackClient, overlay map[string]interface{}, vmConfiguration models.VMConfiguration, ctrlEndpoint string) {
	var switches map[string]models.VM
	var hosts map[string]models.VM
	var defaultOverlayAddress string
	var gatewayIp string

	switches = make(map[string]models.VM)
	hosts = make(map[string]models.VM)

	for sw, connections := range overlay {
		var vmSrc models.VM
		_ = vmSrc
		if val, ok := switches[sw]; ok {
			vmSrc = val
		} else {
			vmObj, err := models.CreateOrFindVM(&osClient, sw, &vmConfiguration)
			log.Printf("New VM %s\n", vmObj.Name)
			if err != nil {
				log.Fatalf("VM creation failed")
			}
			switches[sw] = *vmObj
			vmSrc = *vmObj
			utils.SetOverlayInterface(switches[sw], "")
		}
		connectionsAsserted := connections.([]interface{})
		for _, connection := range connectionsAsserted {
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

			endpoint := connectionString["endpoint"]
			var endpointVM models.VM
			_ = endpointVM

			// Is it a host
			if _, ok := connectionString["ip"]; ok {
				if val, ok := hosts[endpoint]; ok {
					endpointVM = val
				} else {
					vmObject, err := models.CreateOrFindVM(&osClient, endpoint, &vmConfiguration)
					log.Printf("New VM %s\n", vmObject.Name)
					if err != nil {
						log.Fatalf("VM creation failed")
					}
					ip := connectionString["ip"]
					vmObject.OverlayIp = ip
					hosts[endpoint] = *vmObject
					endpointVM = *vmObject
					utils.SetOverlayInterface(hosts[endpoint], ip)
					utils.RunCommand(hosts[endpoint], "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")

					// Setup the NAT and configure iptables
					if _, ok := connectionString["default"]; ok {
						defaultOverlayAddress = ip
						gatewayIp = vmObject.IP[0]
						utils.RunCommand(endpointVM, "sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE")
						utils.RunCommand(endpointVM, "sudo iptables -P FORWARD ACCEPT")
					}
				}
			} else {
				if val, ok := switches[endpoint]; ok {
					endpointVM = val
				} else {
					vmObject, err := models.CreateOrFindVM(&osClient, endpoint, &vmConfiguration)
					log.Printf("New VM %s\n", vmObject.Name)
					if err != nil {
						log.Fatalf("VM creation failed")
					}
					switches[endpoint] = *vmObject
					endpointVM = *vmObject
					utils.SetOverlayInterface(switches[endpoint], "")
				}
			}
			connectNodes(vmSrc, endpointVM)
		}
	}

	log.Println("Replacing default gateway with the new gateway")
	for _, vm := range hosts {
		if vm.OverlayIp != defaultOverlayAddress {
			utils.RunCommand(vm, "sshpass -p savi ssh -o StrictHostKeyChecking=no ubuntu@"+gatewayIp+" ping -c 5 "+vm.OverlayIp)
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

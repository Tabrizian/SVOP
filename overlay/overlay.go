/**
* File              : overlay.go
* Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
* Date              : 10.04.2019
* Last Modified Date: 11.04.2019
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

func DeployOverlay(osClient utils.OpenStackClient, overlay map[string]interface{}, vmConfiguration models.VMConfiguration) {
	var switches map[string]models.VM
	var hosts map[string]models.VM

	switches = make(map[string]models.VM)
	hosts = make(map[string]models.VM)

	for sw, connections := range overlay {
		var vmSrc models.VM
		if val, ok := switches[sw]; ok {
			vmSrc = val
		} else {
			vmObj, err := models.NewVM(&osClient, sw, &vmConfiguration)
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

			// Is it a host
			if _, ok := connectionString["ip"]; ok {
				if val, ok := hosts[endpoint]; ok {
					endpointVM = val
				} else {
					vmObject, err := models.NewVM(&osClient, endpoint, &vmConfiguration)
					log.Printf("New VM %s\n", vmObject.Name)
					if err != nil {
						log.Fatalf("VM creation failed")
					}
					hosts[endpoint] = *vmObject
					endpointVM = *vmObject
					ip := connectionString["ip"]
					utils.SetOverlayInterface(hosts[endpoint], ip)
				}
			} else {
				if val, ok := switches[endpoint]; ok {
					endpointVM = val
				} else {
					vmObject, err := models.NewVM(&osClient, endpoint, &vmConfiguration)
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
}

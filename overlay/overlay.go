/**
 * File              : overlay.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 10.04.2019
 * Last Modified Date: 11.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package overlay

import (
	"log"

	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/utils"
)

func DeployOverlay(osClient utils.OpenStackClient, overlay map[string]interface{}, vmConfiguration models.VMConfiguration) {
	var switches map[string]models.VM
	var hosts map[string]models.VM

	switches = make(map[string]models.VM)
	hosts = make(map[string]models.VM)

	for sw, connections := range overlay {
		var vmObj models.VM
		if val, ok := switches[sw]; ok {
			vmObj = val
		} else {
			vmObj, err := models.NewVM(&osClient, sw, &vmConfiguration)
			log.Printf("New VM %s\n", vmObj.Name)
			if err != nil {
				log.Fatalf("VM creation failed")
			}
			switches[sw] = *vmObj
			utils.SetOverlayInterface(switches[sw], "")
		}
		_ = vmObj
		connectionsAsserted := connections.([]interface{})
		for _, connection := range connectionsAsserted {
			connectionAsserted := connection.(map[interface {}]interface{})
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
					vmObj, err := models.NewVM(&osClient, endpoint, &vmConfiguration)
					log.Printf("New VM %s\n", vmObj.Name)
					if err != nil {
						log.Fatalf("VM creation failed")
					}
					hosts[endpoint] = *vmObj
					ip := connectionString["ip"]
					utils.SetOverlayInterface(hosts[endpoint], ip)
				}
			} else {
				if val, ok := switches[endpoint]; ok {
					endpointVM = val
				} else {
					vmObj, err := models.NewVM(&osClient, endpoint, &vmConfiguration)
					log.Printf("New VM %s\n", vmObj.Name)
					if err != nil {
						log.Fatalf("VM creation failed")
					}
					switches[endpoint] = *vmObj
					utils.SetOverlayInterface(switches[endpoint], "")
				}
			}

			_ = endpointVM
		}
	}
}

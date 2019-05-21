/**
 * File              : app.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 15.04.2019
 * Last Modified Date: 21.05.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package apps

import (
	"fmt"
	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/overlay"
	// "github.com/Tabrizian/SVOP/utils"
	"github.com/pkg/errors"
	"log"
)

func DeployServices(services []interface{}, overlayT *overlay.Overlay) error {
	var gateway overlay.Host
	for _, host := range overlayT.Hosts {
		if host.Gateway == true {
			gateway = host
			break
		}
	}

	log.Println(gateway)

	for _, service := range services {
		serviceAsserted := overlay.FixConnectionFormat(service)
		log.Println(serviceAsserted)
		dst := overlayT.Hosts[serviceAsserted["location"]]
		cmd := fmt.Sprintf("docker run --name %s -d -p %s:%s %s", serviceAsserted["name"], serviceAsserted["inPort"], serviceAsserted["outPort"], serviceAsserted["image"])
		// dstVMObj, err := models.GetVMByName(overlayT.OsClient, dst.Name)
		// if err != nil {
		// 	return errors.Wrap(err, "Couldn't find VM "+dst.Name)
		// }

		gatewayVMObj, err := models.GetVMByName(overlayT.OsClient, gateway.Name)
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+gateway.Name)
		}

		// utils.RunCommandFromOverlay(dst, cmd, gatewayVMObj.IP[0])
		loadBalance := fmt.Sprintf("sudo iptables -t nat -A PREROUTING -d %s/32 -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", gatewayVMObj.IP[0], serviceAsserted["portInLB"], dst.OverlayIP, serviceAsserted["outPort"])
		log.Println(loadBalance)
		log.Println(cmd)

		// utils.RunCommand(gateway, loadBalance)
	}
	return nil
}

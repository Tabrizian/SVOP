/**
 * File              : app.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 15.04.2019
 * Last Modified Date: 23.05.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package apps

import (
	"fmt"
	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/overlay"
	"github.com/Tabrizian/SVOP/utils"
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

	for _, service := range services {
		serviceAsserted := FixServiceFormat(service)
		dst := overlayT.Hosts[serviceAsserted["location"].(string)]
		portString := ""
		dstVMObj, err := models.GetVMByName(overlayT.OsClient, dst.Name)
		dstVMObj.OverlayIp = dst.OverlayIP
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+dst.Name)
		}

		gatewayVMObj, err := models.GetVMByName(overlayT.OsClient, gateway.Name)
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+gatewayVMObj.Name)
		}
		log.Println(serviceAsserted)
		ports := serviceAsserted["ports"].([]interface{})
		for _, port := range ports {
			portAs := port.(map[interface{}]interface{})
			portAsserted := overlay.FixConnectionFormat(portAs)
			portString = portString + fmt.Sprintf(" -p %s:%s", portAsserted["outPort"], portAsserted["inPort"])
			if _, ok := portAsserted["portInLB"]; ok {
				loadBalance := fmt.Sprintf("sudo iptables -t nat -A PREROUTING -d %s/32 -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", gatewayVMObj.IP[0], portAsserted["portInLB"], dst.OverlayIP, portAsserted["outPort"])
				utils.RunCommand(*gatewayVMObj, loadBalance)
			}
		}
		cmd := fmt.Sprintf("docker run --name %s -d %s %s", serviceAsserted["name"], portString, serviceAsserted["image"])

		utils.RunCommandFromOverlay(*dstVMObj, cmd, gatewayVMObj.IP[0])
	}
	return nil
}

func FixServiceFormat(service interface{}) map[string]interface{} {
	serviceAsserted := service.(map[interface{}]interface{})
	serviceString := make(map[string]interface{})
	for key, value := range serviceAsserted {
		switch key := key.(type) {
		case string:
			serviceString[key] = value
		}
	}
	return serviceString
}

func DeleteServices(services []interface{}, overlayT *overlay.Overlay) error {
	var gateway overlay.Host
	for _, host := range overlayT.Hosts {
		if host.Gateway == true {
			gateway = host
			break
		}
	}

	for _, service := range services {
		serviceAsserted := overlay.FixConnectionFormat(service)
		dst := overlayT.Hosts[serviceAsserted["location"]]
		cmd := fmt.Sprintf("docker rm -f %s", serviceAsserted["name"])
		dstVMObj, err := models.GetVMByName(overlayT.OsClient, dst.Name)
		dstVMObj.OverlayIp = dst.OverlayIP
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+dst.Name)
		}

		gatewayVMObj, err := models.GetVMByName(overlayT.OsClient, gateway.Name)
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+gatewayVMObj.Name)
		}

		utils.RunCommandFromOverlay(*dstVMObj, cmd, gatewayVMObj.IP[0])
		loadBalance := fmt.Sprintf("sudo iptables -t nat -D PREROUTING -d %s/32 -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", gatewayVMObj.IP[0], serviceAsserted["portInLB"], dst.OverlayIP, serviceAsserted["outPort"])

		utils.RunCommand(*gatewayVMObj, loadBalance)
	}
	return nil
}

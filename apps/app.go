/**
 * File              : app.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 15.04.2019
 * Last Modified Date: 26.05.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package apps

import (
	"fmt"
	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/overlay"
	"github.com/Tabrizian/SVOP/utils"
	"github.com/pkg/errors"
	"strconv"
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
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+dst.Name)
		}
		dstVMObj.OverlayIp = dst.OverlayIP

		gatewayVMObj, err := models.GetVMByName(overlayT.OsClient, gateway.Name)
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+gatewayVMObj.Name)
		}
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
		portNumber, err := strconv.Atoi(serviceAsserted["metrics"].(string))
		if err != nil {
			return errors.Wrap(err, "Failed to convert metrics port number to int")
		}
		fmt.Println(dst.Name)
		serviceApp := &utils.Service{
			Name:    dst.Name + "-" + serviceAsserted["name"].(string),
			Tags:    []string{"overlay", "service", overlayT.OsClient.Auth.Region},
			Port:    portNumber,
			Address: dstVMObj.IP[0],
		}
		overlayT.ConsulClient.RegisterService(serviceApp)

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
		serviceAsserted := FixServiceFormat(service)
		dst := overlayT.Hosts[serviceAsserted["location"].(string)]
		portString := ""
		cmd := fmt.Sprintf("docker rm -f %s", serviceAsserted["name"])
		dstVMObj, err := models.GetVMByName(overlayT.OsClient, dst.Name)
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+dst.Name)
		}
		dstVMObj.OverlayIp = dst.OverlayIP

		gatewayVMObj, err := models.GetVMByName(overlayT.OsClient, gateway.Name)
		if err != nil {
			return errors.Wrap(err, "Couldn't find VM "+gatewayVMObj.Name)
		}
		utils.RunCommandFromOverlay(*dstVMObj, cmd, gatewayVMObj.IP[0])

		ports := serviceAsserted["ports"].([]interface{})
		for _, port := range ports {
			portAs := port.(map[interface{}]interface{})
			portAsserted := overlay.FixConnectionFormat(portAs)
			portString = portString + fmt.Sprintf(" -p %s:%s", portAsserted["outPort"], portAsserted["inPort"])
			if _, ok := portAsserted["portInLB"]; ok {
				loadBalance := fmt.Sprintf("sudo iptables -t nat -D PREROUTING -d %s/32 -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", gatewayVMObj.IP[0], portAsserted["portInLB"], dst.OverlayIP, portAsserted["outPort"])
				utils.RunCommand(*gatewayVMObj, loadBalance)
			}
		}

	}
	return nil
}

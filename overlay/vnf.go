/**
 * File              : vnf.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 23.04.2019
 * Last Modified Date: 18.06.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package overlay

import (
	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/utils"
	"github.com/pkg/errors"
	"log"
	"strings"
)

func DeployVNFs(overlayObj Overlay, vnfs map[string]interface{}) error {
	osClient := overlayObj.OsClient
	gateway := overlayObj.GetGateway()
	gatewayVMObj, err := models.GetVMByName(osClient, gateway.Name)
	if err != nil {
		return errors.Wrap(err, "Failed to get VM "+gateway.Name)
	}
	ip := gatewayVMObj.IP[0]
	log.Println(ip)
	for _, value := range vnfs {
		valueAsserted := FixConnectionFormat(value)
		location := valueAsserted["location"]
		image := valueAsserted["image"]
		vmOv := overlayObj.Hosts[location]
		log.Println("Deploying VNF " + image + " into the " + location)
		vm, err := models.GetVMByName(osClient, location)
		if err != nil {
			return errors.Wrap(err, "Failed to get VM "+location)
		}
		vmD := *vm
		vmD.OverlayIp = vmOv.OverlayIP
		utils.RunCommandFromOverlay(vmD, "sudo ovs-vsctl add-port br1 ingress -- set interface ingress type=internal", ip)
		utils.RunCommandFromOverlay(vmD, "sudo ifconfig ingress up", ip)
		utils.RunCommandFromOverlay(vmD, "sudo ovs-vsctl add-port br1 egress -- set interface egress type=internal", ip)
		utils.RunCommandFromOverlay(vmD, "sudo ifconfig egress up", ip)
		utils.RunCommandFromOverlay(vmD, "docker run -d --net=host "+image, ip)

		switches, _ := utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl show br1 | awk '/[0-9]\\(.*\\):/ {print}' | awk -F '[()]' '{print $1, $2}'", ip)
		macs, _ := utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl show br1 | awk '/[0-9]\\(.*\\):/ {print}' | awk -F 'addr:' '{print  $2}'", ip)
		switchAndPortNumber := strings.Trim(string(switches), "\n")
		switchAndPortNumber = strings.Trim(switchAndPortNumber, " ")
		macsTrimmed := strings.Trim(string(macs), "\n")
		macsTrimmed = strings.Trim(macsTrimmed, " ")
		macsSplitted := strings.Split(macsTrimmed, "\n")

		var vxlanPortNumber string

		var interalPortNumber string
		var interalIfaceMAC string

		var snortIngress string
		var snortEgress string

		for index, line := range strings.Split(switchAndPortNumber, "\n") {
			lineTrimmed := strings.Trim(line, " ")
			port := strings.Split(lineTrimmed, " ")[0]
			sw := strings.Split(lineTrimmed, " ")[1]
			if sw == "br1-internal" {
				interalPortNumber = port
				interalIfaceMAC = macsSplitted[index]
			} else if sw == "ingress" {
				snortIngress = port
			} else if sw == "egress" {
				snortEgress = port
			} else if strings.HasPrefix(sw, vmD.Name) {
				vxlanPortNumber = port
			}
		}

		utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl add-flow br1 in_port="+vxlanPortNumber+","+"priority=9999,dl_dst="+interalIfaceMAC+
			",actions=output:"+interalPortNumber, ip)
		utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl add-flow br1 in_port="+vxlanPortNumber+","+"priority=9999,dl_dst="+"01:00:00:00:00:00/01:00:00:00:00:00"+
			",actions=output:"+interalPortNumber, ip)
		utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl add-flow br1 in_port="+interalPortNumber+","+"actions=output:"+vxlanPortNumber, ip)
		utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl add-flow br1 in_port="+vxlanPortNumber+","+"priority=4444"+
			",actions=output:"+snortIngress, ip)
		utils.RunCommandFromOverlay(vmD, "sudo ovs-ofctl add-flow br1 in_port="+snortEgress+","+"priority=4444"+
			",actions=output:"+vxlanPortNumber, ip)

	}
	return nil
}

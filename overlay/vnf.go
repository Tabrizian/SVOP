/**
 * File              : vnf.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 23.04.2019
 * Last Modified Date: 23.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package overlay

import (
	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/utils"
	"github.com/pkg/errors"
)

func DeployVNFs(osClient utils.OpenStackClient, vnfs map[string]interface{}) error {
	for _, value := range vnfs {
		valueAsserted := FixConnectionFormat(value)
		location := valueAsserted["location"]
		image := valueAsserted["image"]
		vm, err := models.GetVMByName(&osClient, location)
		if err != nil {
			return errors.Wrap(err, "Failed to get VM "+location)
		}
		vmD := *vm
		utils.RunCommand(vmD, "sudo ovs-vsctl add-port br1 ingress -- set interface ingress type=internal")
		utils.RunCommand(vmD, "sudo ifconfig ingress up")
		utils.RunCommand(vmD, "sudo ovs-vsctl add-port br1 egress -- set interface egress type=internal")
		utils.RunCommand(vmD, "sudo ifconfig egress up")
		utils.RunCommand(vmD, "docker run -d --net=host "+image)
	}
	return nil
}

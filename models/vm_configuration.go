/**
 * File              : vm_configuration.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 02.04.2019
 * Last Modified Date: 17.05.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package models

import ()

type VMConfiguration struct {
	Image    string
	Network  string
	Secgroup string
	Flavor   string
}

func (vmConfiguration *VMConfiguration) GetImage() string {
	return vmConfiguration.Image
}

func (vmConfiguration *VMConfiguration) GetNetwork() string {
	return vmConfiguration.Network
}

func (vmConfiguration *VMConfiguration) GetSecgroup() string {
	return vmConfiguration.Secgroup
}

func (vmConfiguration *VMConfiguration) GetFlavor() string {
	return vmConfiguration.Flavor
}

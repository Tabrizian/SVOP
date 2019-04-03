/**
 * File              : vm_configuration.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 02.04.2019
 * Last Modified Date: 02.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package models

import ()

type VMConfiguration struct {
    image string
    network string
    secgroup string
}

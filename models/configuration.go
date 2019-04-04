/**
 * File              : configuration.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 01.04.2019
 * Last Modified Date: 03.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package models

import ()

type Configuration struct {
    Auth AuthConfiguration
    VM   VMConfiguration
}


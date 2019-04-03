/**
 * File              : auth_configuration.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 02.04.2019
 * Last Modified Date: 03.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package models

import ()

type AuthConfiguration struct {
    Username string
    Password string
    Region string
    Url string
    Project string
}

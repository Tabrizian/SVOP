/**
 * File              : openstack.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 01.04.2019
 * Last Modified Date: 03.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package utils

import (
  "fmt"
  "net/http"
  "bytes"
  "log"
  "io/ioutil"
  "encoding/json"

  "github.com/Tabrizian/SVOP/models"
)

type OpenStackClient struct {
    AuthToken string
    Auth models.AuthConfiguration
    NovaURL string
}


func NewOpenStackClient(auth models.AuthConfiguration) (*OpenStackClient, error) {
    authBody := fmt.Sprintf("{\"auth\": { " +
		"\"tenantName\": \"%s\"," +
		"\"passwordCredentials\": {" +
        "\"username\": \"%s\"," +
        "\"password\": \"%s\" }}}",
        auth.Project, auth.Username, auth.Password)

    resp, err := http.Post(
        auth.Url + "/tokens",
        "application/json",
        bytes.NewReader([]byte(authBody)),
    )

    if err != nil {
        log.Fatalf("Authentication failed because %s", err)
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Fatalf("Failed to read all of the response %s", err)
    }

    var result interface{}
    err = json.Unmarshal(body, &result)
    if err != nil {
        log.Fatalf("Failed to read all of the response %s", err)
    }
    resultAsserted := result.(map[string]interface{})
    access := resultAsserted["access"].(map[string]interface{})
    token := access["token"].(map[string]interface{})

    osClient := &OpenStackClient{
        AuthToken: token["id"].(string),
        Auth: auth,
    }

    serviceCatalog := access["serviceCatalog"].([]interface{})
    for _, catalog := range serviceCatalog {
        catalogAsserted := catalog.(map[string]interface{})
        catalogType := catalogAsserted["type"].(string)
        if catalogType == "compute" {
            endpoints := catalogAsserted["endpoints"].([]interface{})
            for _, endpoint := range endpoints {
                endpointAsserted := endpoint.(map[string]interface{})
                region := endpointAsserted["region"].(string)
                if region == auth.Region {
                    osClient.NovaURL = endpointAsserted["publicURL"].(string)
                }
            }
        }
    }

    return osClient, nil
}


func (osClient *OpenStackClient) GetAuthToken() {
    log.Print(osClient.AuthToken)
}

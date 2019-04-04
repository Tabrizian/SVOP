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
    NetworkURL string
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
        if catalogType == "network" {
            endpoints := catalogAsserted["endpoints"].([]interface{})
            for _, endpoint := range endpoints {
                endpointAsserted := endpoint.(map[string]interface{})
                region := endpointAsserted["region"].(string)
                if region == auth.Region {
                    osClient.NetworkURL = endpointAsserted["publicURL"].(string)
                }
            }
        }
    }

    return osClient, err
}

func authRequest(verb string, url string, body string, authToken string) []byte {
    req, err := http.NewRequest(verb, url, bytes.NewReader([]byte(body)))
    if verb == "POST" {
        req, err = http.NewRequest(verb, url, bytes.NewReader([]byte(body)))
        req.Header.Add("Content-Type", "application/json")
    } else {
        req, err = http.NewRequest(verb, url, nil)
    }

    if err != nil {
        log.Fatalf("An error occured in creation of new request %s", err)
    }

    client := &http.Client{}
    req.Header.Add("X-Auth-Token", authToken)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("An error occured in fetching the result - %s", err)
    }

    defer resp.Body.Close()
    bodyByte, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("An error occurred in reading all of the error response body - %s", err)
    }

    return bodyByte
}

func (osClient *OpenStackClient) GetAuthToken() {
}

func (osClient *OpenStackClient) GetFlavorID(flavor string) string {

    body := authRequest("GET", osClient.NovaURL + "/flavors", "", osClient.AuthToken)

    var result interface{}
    err := json.Unmarshal(body, &result)
    if err != nil {
        log.Fatalf("Failed to read all of the response %s", err)
    }

    resultAsserted := result.(map[string]interface{})
    flavors := resultAsserted["flavors"].([]interface{})
    flavorId := ""

    for _, flavorItem := range flavors {
        flavorAsserted := flavorItem.(map[string]interface{})
        flavorName := flavorAsserted["name"].(string)
        if flavorName == flavor {
            flavorId = flavorAsserted["id"].(string)
            break
        }
    }

    return flavorId
}

func (osClient *OpenStackClient) GetNetworkID(network string) string {

    body := authRequest("GET", osClient.NetworkURL + "/v2.0/networks", "", osClient.AuthToken)

    var result interface{}
    err := json.Unmarshal(body, &result)
    if err != nil {
        log.Fatalf("Failed to read all of the response %s", err)
    }

    resultAsserted := result.(map[string]interface{})
    networks := resultAsserted["networks"].([]interface{})
    networkId := ""

    for _, networkItem := range networks {
        networkAsserted := networkItem.(map[string]interface{})
        networkName := networkAsserted["name"].(string)
        if networkName == network {
            networkId = networkAsserted["id"].(string)
            break
        }
    }

    return networkId
}

func (osClient *OpenStackClient) GetImageID(image string) string {

    body := authRequest("GET", osClient.NovaURL + "/images", "", osClient.AuthToken)

    var result interface{}
    err := json.Unmarshal(body, &result)
    if err != nil {
        log.Fatalf("Failed to read all of the response %s", err)
    }

    resultAsserted := result.(map[string]interface{})
    images := resultAsserted["images"].([]interface{})
    imageId := ""

    for _, imageItem := range images {
        imageAsserted := imageItem.(map[string]interface{})
        imageName := imageAsserted["name"].(string)
        if imageName == image {
            imageId = imageAsserted["id"].(string)
            break
        }
    }

    return imageId
}

func (osClient *OpenStackClient) GetSecgroupID(secgroup string) string {

    body := authRequest("GET", osClient.NetworkURL + "/v2.0/security-groups", "", osClient.AuthToken)

    var result interface{}
    err := json.Unmarshal(body, &result)
    if err != nil {
        log.Fatalf("Failed to read all of the response %s", err)
    }

    resultAsserted := result.(map[string]interface{})
    secgroups := resultAsserted["security_groups"].([]interface{})
    secgroupId := ""

    for _, secgroupItem := range secgroups {
        secgroupAsserted := secgroupItem.(map[string]interface{})
        secgroupName := secgroupAsserted["name"].(string)
        if secgroupName == secgroup {
            secgroupId = secgroupAsserted["id"].(string)
            break
        }
    }

    return secgroupId
}

func (osClient *OpenStackClient) CreateServer(name string, vmConfiguration models.VMConfiguration) {
    flavor := osClient.GetFlavorID(vmConfiguration.Flavor)
    image := osClient.GetImageID(vmConfiguration.Image)
    network := osClient.GetNetworkID(vmConfiguration.Network)
    secgroup := osClient.GetSecgroupID(vmConfiguration.Secgroup)
    reqBody := fmt.Sprintf("{ \"server\": { \"name\": \"%s\", " +
        "\"flavorRef\": \"%s\", \"imageRef\": \"%s\", " +
        "\"networks\": [{ \"uuid\": \"%s\"}], \"security_groups\": [{" +
        "\"name\": \"%s\" }]}}", name, flavor, image, network, secgroup)

    body := authRequest("POST", osClient.NovaURL + "/servers", reqBody, osClient.AuthToken)
    log.Print(osClient.NovaURL + "/servers")
    log.Print(string(body))
}

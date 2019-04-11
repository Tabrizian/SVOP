/**
* File              : openstack.go
* Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
* Date              : 01.04.2019
* Last Modified Date: 11.04.2019
* Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Tabrizian/SVOP/models"
)

type OpenStackClient struct {
	AuthToken  string
	Auth       models.AuthConfiguration
	NovaURL    string
	NetworkURL string
}

func NewOpenStackClient(auth models.AuthConfiguration) (*OpenStackClient, error) {
	authBody := fmt.Sprintf("{\"auth\": { "+
		"\"tenantName\": \"%s\","+
		"\"passwordCredentials\": {"+
		"\"username\": \"%s\","+
		"\"password\": \"%s\" }}}",
		auth.Project, auth.Username, auth.Password)

	resp, err := http.Post(
		auth.Url+"/tokens",
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
		Auth:      auth,
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

func (osClient *OpenStackClient) GetFlavorID(flavor string) string {

	body := AuthRequest("GET", osClient.NovaURL+"/flavors", "", osClient.AuthToken)

	var result interface{}
	err := json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Failed to parse json - %s", err)
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

	body := AuthRequest("GET", osClient.NetworkURL+"/v2.0/networks", "", osClient.AuthToken)

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

	body := AuthRequest("GET", osClient.NovaURL+"/images", "", osClient.AuthToken)

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

	body := AuthRequest("GET", osClient.NetworkURL+"/v2.0/security-groups", "", osClient.AuthToken)

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

func (osClient *OpenStackClient) CreateServer(name string, vmConfiguration models.VMConfiguration) *models.VM {
	flavor := osClient.GetFlavorID(vmConfiguration.Flavor)
	image := osClient.GetImageID(vmConfiguration.Image)
	network := osClient.GetNetworkID(vmConfiguration.Network)
	secgroup := osClient.GetSecgroupID(vmConfiguration.Secgroup)
	reqBody := fmt.Sprintf("{ \"server\": { \"name\": \"%s\", "+
		"\"flavorRef\": \"%s\", \"imageRef\": \"%s\", "+
		"\"networks\": [{ \"uuid\": \"%s\"}], \"security_groups\": [{"+
		"\"name\": \"%s\" }]}}", name, flavor, image, network, secgroup)

	body := AuthRequest("POST", osClient.NovaURL+"/servers", reqBody, osClient.AuthToken)

	var result interface{}
	err := json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Failed to read all of the response %s", err)
	}

	resultAsserted := result.(map[string]interface{})
	server := resultAsserted["server"]
	serverAsserted := server.(map[string]interface{})
	id := serverAsserted["id"].(string)
	fmt.Print(id)

	vm := &models.VM{
		Name: name,
		Id:   id,
	}

	return vm
}

func (osClient *OpenStackClient) GetAuthToken() string {
	return osClient.AuthToken
}

func (osClient *OpenStackClient) GetNovaURL() string {
	return osClient.NovaURL
}

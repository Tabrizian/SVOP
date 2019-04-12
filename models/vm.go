/**
* File              : vm.go
* Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
* Date              : 04.04.2019
* Last Modified Date: 12.04.2019
* Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type IOpenStackClient interface {
	GetAuthToken() string
	GetNovaURL() string
	GetFlavorID(string) string
	GetImageID(string) string
	GetNetworkID(string) string
	GetSecgroupID(string) string
}

type IVMConfiguration interface {
	GetImage() string
	GetNetwork() string
	GetSecgroup() string
	GetFlavor() string
}

type VM struct {
	Name     string
	IP       []string
	Id       string
	OsClient IOpenStackClient
}

func AuthRequest(verb string, url string, body string, authToken string) []byte {
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

func NewVM(osClient IOpenStackClient, name string, vmConfiguration IVMConfiguration) (*VM, error) {
	flavor := osClient.GetFlavorID(vmConfiguration.GetFlavor())
	image := osClient.GetImageID(vmConfiguration.GetImage())
	network := osClient.GetNetworkID(vmConfiguration.GetNetwork())
	secgroup := osClient.GetSecgroupID(vmConfiguration.GetSecgroup())
	reqBody := fmt.Sprintf("{ \"server\": { \"name\": \"%s\", "+
		"\"flavorRef\": \"%s\", \"imageRef\": \"%s\", "+
		"\"networks\": [{ \"uuid\": \"%s\"}], \"security_groups\": [{"+
		"\"name\": \"%s\" }]}}", name, flavor, image, network, secgroup)

	body := AuthRequest("POST", osClient.GetNovaURL()+"/servers", reqBody, osClient.GetAuthToken())

	var result interface{}
	err := json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Failed to read all of the response %s", err)
	}

	resultAsserted := result.(map[string]interface{})
	server := resultAsserted["server"]
	serverAsserted := server.(map[string]interface{})
	id := serverAsserted["id"].(string)

	vm := &VM{
		Name: name,
		Id:   id,
	}
	vm.OsClient = osClient
	vm.WaitForIP()
	vm.Update()

	return vm, err
}

func GetVM(osClient IOpenStackClient, id string) (*VM, error) {
	vm := &VM{
		Id: id,
	}
	vm.OsClient = osClient
	resp := vm.RefreshVM()
	var result map[string]interface{}
	err := json.Unmarshal(resp, &result)
	if err != nil {
		log.Fatalf("Failed to parse JSON - %s", err)
	}
	server := result["server"].(map[string]interface{})
	addresses := server["addresses"].(map[string]interface{})

	var ips []string

	for _, interfaceI := range addresses {
		interfaceAsserted := interfaceI.([]interface{})
		ip := interfaceAsserted[0].(map[string]interface{})

		ips = append(ips, (ip["addr"].(string)))
	}

	vm.IP = ips

	return vm, err
}

func GetVMByName(osClient IOpenStackClient, name string) (*VM, error) {
	resp := AuthRequest("GET", osClient.GetNovaURL()+"/servers?name=^"+name, "", osClient.GetAuthToken())
	var result map[string]interface{}
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse JSON")
	}

	servers := result["servers"].([]interface{})
	if len(servers) > 1 {
		return nil, errors.New("More than one server with this name exists")
	} else if len(servers) == 0 {
		return nil, errors.New("No server with this name exist")
	}

	server := servers[0].(map[string]interface{})
	id := server["id"].(string)

	return GetVM(osClient, id)
}

func (vm *VM) RefreshVM() []byte {
	osClient := vm.OsClient
	auth := osClient.GetAuthToken()
	url := osClient.GetNovaURL()
	body := AuthRequest("GET", url+"/servers/"+vm.Id, "", auth)
	return body
}

func (vm *VM) Update() {
	resp := vm.RefreshVM()
	var result map[string]interface{}
	err := json.Unmarshal(resp, &result)
	if err != nil {
		log.Fatalf("Failed to parse JSON - %s", err)
	}
	server := result["server"].(map[string]interface{})
	addresses := server["addresses"].(map[string]interface{})

	var ips []string

	for _, interfaceI := range addresses {
		interfaceAsserted := interfaceI.([]interface{})
		ip := interfaceAsserted[0].(map[string]interface{})

		ips = append(ips, (ip["addr"].(string)))
	}

	vm.IP = ips
}

func (vm *VM) WaitForIP() {
	for {
		resp := vm.RefreshVM()
		time.Sleep(1000 * time.Millisecond)

		var result interface{}
		err := json.Unmarshal(resp, &result)

		if err != nil {
			log.Fatalf("Failed to parse JSON - %s", err)
		}

		resultAsserted := result.(map[string]interface{})
		server := resultAsserted["server"].(map[string]interface{})
		addresses := server["addresses"].(map[string]interface{})
		status := server["status"].(string)

		if len(addresses) > 0 && status == "ACTIVE" {
			vm.Update()
			break
		}
	}
}

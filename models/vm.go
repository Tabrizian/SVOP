/**
 * File              : vm.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 04.04.2019
 * Last Modified Date: 07.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package models

import (
)

import (
  "net/http"
  "log"
  "io/ioutil"
  "bytes"
  "encoding/json"
  "time"
  "fmt"
)

type IOpenStackClient interface {
    GetAuthToken() string
    GetNovaURL() string
    GetFlavorID() string
    GetImageID() string
}

type IVMConfiguration interface {
    GetImage() string
    GetNetwork() string
    GetSecgroup() string
    GetFlavor() string
}

type VM struct {
    Name string
    IP []string
    Id string
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

func NewVM(osClient IOpenStackClient, vmConfiguration IVMConfiguration) (*VM, error) {
    flavor := osClient.GetFlavorID(vmConfiguration.GetFlavor())
    image := osClient.GetImageID(vmConfiguration.GetImage())
    network := osClient.GetNetworkID(vmConfiguration.GetNetwork())
    secgroup := osClient.GetSecgroupID(vmConfiguration.GetSecgroup())
    reqBody := fmt.Sprintf("{ \"server\": { \"name\": \"%s\", " +
        "\"flavorRef\": \"%s\", \"imageRef\": \"%s\", " +
        "\"networks\": [{ \"uuid\": \"%s\"}], \"security_groups\": [{" +
        "\"name\": \"%s\" }]}}", name, flavor, image, network, secgroup)

    body := AuthRequest("POST", osClient.GetNovaURL() + "/servers", reqBody, osClient.GetAuthToken())

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
        Id: id,
    }

    return vm
}


func (vm *VM) RefreshVM(osClient IOpenStackClient) []byte {
    auth := osClient.GetAuthToken()
    url := osClient.GetNovaURL()
    body := AuthRequest("GET", url + "/servers/" + vm.Id, "", auth)
    return body
}

func (vm *VM) Update(osClient IOpenStackClient) {
    resp := vm.RefreshVM(osClient)
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

func (vm *VM) WaitForIP(osClient IOpenStackClient) {
    for {
        resp := vm.RefreshVM(osClient)
        time.Sleep(1000 * time.Millisecond)

        var result interface{}
        err := json.Unmarshal(resp, &result)

        if err != nil {
            log.Fatalf("Failed to parse JSON - %s", err)
        }

        resultAsserted := result.(map[string]interface{})
        server := resultAsserted["server"].(map[string]interface{})
        addresses := server["addresses"].(map[string]interface{})

        if len(addresses) > 0 {
            vm.Update(osClient)
            break
        }
    }
}

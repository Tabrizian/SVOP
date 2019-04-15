/**
 * File              : ryu.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 14.04.2019
 * Last Modified Date: 14.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package utils

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
)

type RyuClient struct {
	URL string
}

type RyuPort struct {
	Hw_addr string
	Name    string
	Port_no string
	Dpid    string
}

type RyuSwitch struct {
	Ports []RyuPort
	Dpid  string
}

func NewRyuClient(URL string) (*RyuClient, error) {
	ryuClient := &RyuClient{
		URL: "http://" + URL,
	}

	return ryuClient, nil
}

func request(verb string, url string, body string) []byte {
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

// Returns the list of all of the switches
func (ryuClient *RyuClient) GetSwitches() ([]RyuSwitch, error) {
	resp := request("GET", ryuClient.URL+"/v1.0/topology/switches", "")

	var result []RyuSwitch
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse JSON")
	}

	return result, nil
}

func (ryuClient *RyuClient) CreatePath(src string, dst string) {
	connections := make(map[string][]string)
	graph := make(map[string][]string)
	switches, _ := ryuClient.GetSwitches()
	for _, sw := range switches {
		for _, port := range sw.Ports {
			graph[sw.Dpid] = append(graph[sw.Dpid], sw.Dpid+"/"+port.Port_no)
			parts := strings.Split(port.Name, "-")
			graph[sw.Dpid+"/"+port.Port_no] = append(graph[sw.Dpid+"/"+port.Port_no], sw.Dpid)
			sort.Strings(parts)
			connections[parts[0]+"-"+parts[1]] = append(connections[parts[0]+"-"+parts[1]], sw.Dpid+"/"+port.Port_no)
		}
	}

	for node, connection := range connections {
		src := strings.Split(node, "-")[0]
		if src[0] != 'h' {
			graph[connection[0]] = append(graph[connection[0]], connection[1])
			graph[connection[1]] = append(graph[connection[1]], connection[0])
		}
	}

	log.Println(graph)
}

/**
 * File              : ryu.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 14.04.2019
 * Last Modified Date: 25.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package utils

import (
	"bytes"
	"encoding/json"
	"github.com/RyanCarrier/dijkstra"
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
	HWAddr string `json:"hw_addr"`
	Name   string `json:"name"`
	PortNo string `json:"port_no"`
	DPid   string `json:"dpid"`
}

type RyuSwitch struct {
	Ports []RyuPort
	Dpid  string
}

type Match struct {
	DLSrc   string `json:"dl_src,omitempty"`
	DLDst   string `json:"dl_dst,omitempty"`
	InPort  int    `json:"in_port,omitempty"`
	DLType  int    `json:"dl_type,omitempty"`
	NWSrc   string `json:"nw_src,omitempty"`
	NWDst   string `json:"nw_dst,omitempty"`
	NWProto int    `json:"nw_proto,omitempty"`
	TPDst   int    `json:"tp_dst,omitempty"`
}

type PortAction struct {
	Port int    `json:"port"`
	Type string `json:"type"`
}

type Rule struct {
	Matching Match        `json:"match"`
	Action   []PortAction `json:"actions"`
	Dpid     int          `json:"dpid"`
	Priority int          `json:"priority,omitempty"`
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

func (ryuClient *RyuClient) FindShortestPath(src string, dst string) []string {
	connections := make(map[string][]string)
	graph := make(map[string][]string)
	switches, _ := ryuClient.GetSwitches()
	var srcAddress string
	var dstAddress string
	for _, sw := range switches {
		for _, port := range sw.Ports {
			graph[sw.Dpid] = append(graph[sw.Dpid], sw.Dpid+"/"+port.PortNo)
			parts := strings.Split(port.Name, "-")
			graph[sw.Dpid+"/"+port.PortNo] = append(graph[sw.Dpid+"/"+port.PortNo], sw.Dpid)
			sort.Strings(parts)
			connections[parts[0]+"-"+parts[1]] = append(connections[parts[0]+"-"+parts[1]], sw.Dpid+"/"+port.PortNo)
		}
	}

	for node, connection := range connections {
		srcH := strings.Split(node, "-")[0]
		if srcH[0] != 'h' {
			graph[connection[0]] = append(graph[connection[0]], connection[1])
			graph[connection[1]] = append(graph[connection[1]], connection[0])
		} else {
			if srcH == src {
				srcAddress = connection[0]
			} else if srcH == dst {
				dstAddress = connection[0]
			}
		}
	}

	var nodes map[string]int
	var nodesDecode map[int]string
	nodes = make(map[string]int)
	nodesDecode = make(map[int]string)
	graphDJ := dijkstra.NewGraph()
	index := 0
	for vertex, _ := range graph {
		nodes[vertex] = index
		graphDJ.AddVertex(index)
		index = index + 1
	}

	for vertex, connections := range graph {
		for _, connection := range connections {
			graphDJ.AddArc(nodes[vertex], nodes[connection], 1)
		}
	}

	var bestPath []string

	best, err := graphDJ.Shortest(nodes[srcAddress], nodes[dstAddress])
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range nodes {
		nodesDecode[v] = k
	}

	for _, item := range best.Path {
		bestPath = append(bestPath, nodesDecode[item])
	}

	return bestPath
}

func (ryuClient *RyuClient) InstallFlow(rule Rule) ([]byte, error) {
	bytes, err := json.Marshal(rule)
	if err != nil {
		return nil, errors.Wrap(err, "JSON parsing failed")
	}

	resp := request("POST", ryuClient.URL+"/stats/flowentry/add", string(bytes))
	return resp, nil
}

func (ryuClient *RyuClient) DeleteFlow(rule Rule) ([]byte, error) {
	bytes, err := json.Marshal(rule)
	if err != nil {
		return nil, errors.Wrap(err, "JSON parsing failed")
	}

	resp := request("POST", ryuClient.URL+"/stats/flowentry/delete_strict", string(bytes))
	return resp, nil
}

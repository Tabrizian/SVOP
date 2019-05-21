/**
 * File              : consul.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 29.04.2019
 * Last Modified Date: 21.05.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package utils

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type ConsulClient struct {
	URL string
}

type Service struct {
	Address string   `json:"Address"`
	Name    string   `json:"Name"`
	Port    int      `json:"Port"`
	Tags    []string `json:"Tags"`
}

func NewConsulClient(url string) (*ConsulClient, error) {
	consulClient := &ConsulClient{
		URL: "http://" + url,
	}

	return consulClient, nil
}

func (consulClient *ConsulClient) RegisterService(service *Service) ([]byte, error) {
	requestString, err := json.Marshal(service)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert json to error")
	}
	resp := request("PUT", consulClient.URL+"/v1/agent/service/register", string(requestString))
	return resp, nil
}

func (consulClient *ConsulClient) DeleteService(service *Service) ([]byte, error) {
	resp := request("PUT", consulClient.URL+"/v1/agent/service/deregister/"+service.Name, "")
	return resp, nil
}

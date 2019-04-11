/**
 * File              : main.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 07.04.2019
 * Last Modified Date: 11.04.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */
package main

import (
	"io/ioutil"
	"log"

	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/overlay"
	"github.com/Tabrizian/SVOP/utils"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func main() {
	var configuration models.Configuration
	log.Println("SVOP started")
	viper.SetConfigName("config")
	viper.AddConfigPath("./configs/")
	viper.AddConfigPath("/etc/SVOP/")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		log.Fatalf("Error while unmarshaling the configuration file %s", err)
	}

	var result map[string]interface{}
	buffer, err := ioutil.ReadFile("./configs/topology.yaml")
	if err != nil {
		log.Fatalf("An error occured while reading from file: %s", err)
	}

	osClient, _ := utils.NewOpenStackClient(configuration.Auth)

	err = yaml.Unmarshal(buffer, &result)
	if err != nil {
		log.Fatalf("An error occured while parsing YAML: %s", err)
	}

	osClientD := *osClient

	overlay.DeployOverlay(osClientD, result, configuration.VM)
}

/**
 * File              : main.go
 * Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
 * Date              : 29.04.2019
 * Last Modified Date: 18.06.2019
 * Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/Tabrizian/SVOP/apps"
	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/overlay"
	"github.com/Tabrizian/SVOP/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

var (
	configuration models.Configuration
	osClient      *utils.OpenStackClient
	osClientD     utils.OpenStackClient
	ryuClient     *utils.RyuClient
	consulClient  *utils.ConsulClient
	overlayObj    *overlay.Overlay
	result        map[string]interface{}
)

func ReadInitialConfiguration(topology string, config string) {
	viper.SetConfigName(config)
	viper.AddConfigPath("./configs/")
	viper.AddConfigPath("/etc/SVOP/")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		log.Fatalf("Error while unmarshaling the configuration file %s", err)
	}
	osClient, _ := utils.NewOpenStackClient(configuration.Auth)
	osClientD = *osClient
	ryuClient, err = utils.NewRyuClient(viper.Get("controller.address").(string) + ":8080")
	if err != nil {
		log.Fatalf("An error occurred while creating RyuClient: %s", err)
	}

	consulClient, err = utils.NewConsulClient(viper.Get("consul.address").(string))
	if err != nil {
		log.Fatalf("An error occurred while creating Consul Client: %s", err)
	}

	buffer, err := ioutil.ReadFile(topology)
	if err != nil {
		log.Fatalf("An error occured while reading from file: %s", err)
	}
	err = yaml.Unmarshal(buffer, &result)
	if err != nil {
		log.Fatalf("An error occured while parsing YAML: %s", err)
	}
	overlayObj = overlay.NewOverlay(result, ryuClient, osClient, consulClient)
}

func main() {
	app := cli.NewApp()
	app.Name = "svop"
	app.Usage = "Simple VNF Orchestration Platform"
	app.Action = func(c *cli.Context) error {
		fmt.Println("SVOP: Simple VNF Orchestration Platform")
		return nil
	}

	var topology string
	var config string
	var appsConfig string
	var vnf string
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "topology",
			Value:       "./configs/topology.yaml",
			Usage:       "Topology File Location",
			Destination: &topology,
		},
		cli.StringFlag{
			Name:        "config",
			Value:       "config",
			Usage:       "Configuration File Name",
			Destination: &config,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "deploy",
			Aliases: []string{"d"},
			Usage:   "Deploy A SVOP Object",
			Action: func(c *cli.Context) error {
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name: "vnf",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "file",
							Value:       "./configs/vnfs.yaml",
							Usage:       "VNF Configuration place",
							Destination: &vnf,
						},
					},
					Usage: "Deploys a VNF application",
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						var result map[string]interface{}
						buffer, err := ioutil.ReadFile(vnf)
						if err != nil {
							return errors.Wrap(err, "An error occured while reading from file")
						}
						err = yaml.Unmarshal(buffer, &result)
						if err != nil {
							return errors.Wrap(err, "An error occured while parsing YAML")
						}
						overlay.DeployVNFs(*overlayObj, result)
						return nil
					},
				},
				{
					Name:  "app",
					Usage: "Deploys an Application",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "file",
							Value:       "./configs/apps.yaml",
							Usage:       "Apps File Location",
							Destination: &appsConfig,
						},
					},
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						fmt.Println("Deploy an application")
						var result []interface{}
						buffer, err := ioutil.ReadFile(appsConfig)
						if err != nil {
							return errors.Wrap(err, "An error occured while reading from file")
						}
						err = yaml.Unmarshal(buffer, &result)
						apps.DeployServices(result, overlayObj)
						if err != nil {
							return errors.Wrap(err, "An error occured while parsing YAML")
						}

						return nil
					},
				},
				{
					Name:  "network",
					Usage: "Setups Required OpenFlow Rules",
					Action: func(c *cli.Context) error {
						var result map[string]interface{}
						ReadInitialConfiguration(topology, config)
						buffer, err := ioutil.ReadFile(topology)
						if err != nil {
							return errors.Wrap(err, "An error occured while reading from file")
						}
						err = yaml.Unmarshal(buffer, &result)
						if err != nil {
							return errors.Wrap(err, "An error occured while parsing YAML")
						}

						overlayObj := overlay.NewOverlay(result, ryuClient, osClient, consulClient)

						match := utils.Match{
							InPort: 1,
							DLType: 0x800,
							// 		NWSrc:   "192.168.200.101",
							NWDst:   "192.168.200.102",
							NWProto: 6,
							TPDst:   80,
						}

						action := utils.PortAction{
							Port: 3,
							Type: "OUTPUT",
						}

						var portActions []utils.PortAction
						portActions = append(portActions, action)

						rule := utils.Rule{
							Matching: match,
							Action:   portActions,
							Dpid:     103448656425807,
							Priority: 34000,
						}

						// Redirect UDP traffic destined to h2 first to h3
						// overlayObj.UninstallOpenFlowRules(rule, "h1", "h3")
						arg1 := c.Args().Get(0)
						arg2 := c.Args().Get(1)
						err = overlayObj.SetupOpenFlowRules(rule, arg1, arg2)
						log.Println("Setting up flow rules to redirect traffic from " + arg1 + " to " + arg2)
						if err != nil {
							return errors.Wrap(err, "An error occured while setting up rules")
						}
						return nil
					},
				},
				{
					Name:  "topology",
					Usage: "Deploys an Overlay topology",
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						overlayObj.DeployOverlay(osClientD, result, configuration.VM, viper.Get("controller.address").(string))
						return nil
					},
				},
				{
					Name:  "test",
					Usage: "Test",
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						var result map[string]interface{}
						buffer, err := ioutil.ReadFile(topology)
						if err != nil {
							return errors.Wrap(err, "An error occured while reading from file")
						}
						err = yaml.Unmarshal(buffer, &result)
						if err != nil {
							return errors.Wrap(err, "An error occured while parsing YAML")
						}
						serviceCadvisor := &utils.Service{
							Name:    "fake" + "-cadvisor",
							Tags:    []string{"overlay", "cadvisor"},
							Port:    8080,
							Address: "1.2.3.4",
						}
						result2, err := consulClient.RegisterService(serviceCadvisor)
						log.Println(string(result2))
						return err
					},
				},
			},
		},
		{
			Name:    "remove",
			Aliases: []string{"r"},
			Usage:   "Remove A SVOP Object",
			Action: func(c *cli.Context) error {
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:  "flows",
					Usage: "Delete all of the flows",
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						ryuClient.DeleteAll()
						return nil
					},
				},
				{
					Name:  "app",
					Usage: "Delete Currently Deployed Apps",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "file",
							Value:       "./configs/apps.yaml",
							Usage:       "Apps File Location",
							Destination: &appsConfig,
						},
					},
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						fmt.Println("Deploy an application")
						var result []interface{}
						buffer, err := ioutil.ReadFile(appsConfig)
						if err != nil {
							return errors.Wrap(err, "An error occured while reading from file")
						}
						err = yaml.Unmarshal(buffer, &result)
						apps.DeleteServices(result, overlayObj)
						if err != nil {
							return errors.Wrap(err, "An error occured while parsing YAML")
						}

						return nil
					},
				},
				{
					Name:  "services",
					Usage: "Deregister All Services in Consul",
					Action: func(c *cli.Context) error {
						ReadInitialConfiguration(topology, config)
						for _, host := range overlayObj.Hosts {
							serviceCadvisor := &utils.Service{
								Name: host.Name + "-cadvisor",
							}
							serviceExporter := &utils.Service{
								Name: host.Name + "-prometheus",
							}
							_, err := consulClient.DeleteService(serviceCadvisor)
							if err != nil {
								return errors.Wrap(err, "An error occured while deregistering service")
							}
							_, err = consulClient.DeleteService(serviceExporter)
							if err != nil {
								return errors.Wrap(err, "An error occured while deregistering service")
							}
						}
						for _, sw := range overlayObj.Switches {
							serviceCadvisor := &utils.Service{
								Name: sw.Name + "-cadvisor",
							}
							serviceExporter := &utils.Service{
								Name: sw.Name + "-prometheus",
							}
							_, err := consulClient.DeleteService(serviceCadvisor)
							if err != nil {
								return errors.Wrap(err, "An error occured while deregistering service")
							}
							_, err = consulClient.DeleteService(serviceExporter)
							if err != nil {
								return errors.Wrap(err, "An error occured while deregistering service")
							}
						}
						return nil
					},
				},
			},
		},
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

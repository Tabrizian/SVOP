/**
* File              : overlay.go
* Author            : Iman Tabrizian <iman.tabrizian@gmail.com>
* Date              : 10.04.2019
* Last Modified Date: 23.05.2019
* Last Modified By  : Iman Tabrizian <iman.tabrizian@gmail.com>
 */

package overlay

import (
	"fmt"
	"log"

	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Tabrizian/SVOP/models"
	"github.com/Tabrizian/SVOP/utils"
	"github.com/pkg/errors"
)

var vnis []int

type Host struct {
	Gateway   bool
	OverlayIP string
	Name      string
}

type Switch struct {
	Name string
}

type Overlay struct {
	overlayObject map[string]interface{}
	ryuClient     *utils.RyuClient
	Hosts         map[string]Host
	Switches      map[string]Switch
	OsClient      *utils.OpenStackClient
	consulClient  *utils.ConsulClient
}

func NewOverlay(overlayObject map[string]interface{}, ryuClient *utils.RyuClient, osClient *utils.OpenStackClient, consulClient *utils.ConsulClient) *Overlay {
	overlay := &Overlay{
		overlayObject: overlayObject,
		ryuClient:     ryuClient,
		OsClient:      osClient,
		consulClient:  consulClient,
	}

	overlay.Hosts = ExtractHosts(overlay.overlayObject)
	overlay.Switches = ExtractSWs(overlay.overlayObject)

	return overlay
}

func generateVNI() int {
	vni := rand.Intn(6000)

	for {
		exist := false
		for item := range vnis {
			if item == vni {
				exist = true
				break
			}
		}
		if exist == false {
			vnis = append(vnis, vni)
			break
		}
		vni = rand.Intn(6000)
	}

	return vni
}

func connectNodes(node1 models.VM, node2 models.VM) {
	log.Printf("Making VXLAN between %s and %s\n", node1.Name, node2.Name)

	vni := generateVNI()
	cmd := fmt.Sprintf("sudo ovs-vsctl add-port br1 %s -- set interface %s type=vxlan options:remote_ip=%s options:key=%v", node1.Name+"-"+node2.Name, node1.Name+"-"+node2.Name, node2.IP[0], vni)
	utils.RunCommand(node1, cmd)
	cmd = fmt.Sprintf("sudo ovs-vsctl add-port br1 %s -- set interface %s type=vxlan options:remote_ip=%s options:key=%v", node2.Name+"-"+node1.Name, node2.Name+"-"+node1.Name, node1.IP[0], vni)
	utils.RunCommand(node2, cmd)
}

func FixConnectionFormat(connection interface{}) map[string]string {
	connectionAsserted := connection.(map[interface{}]interface{})
	connectionString := make(map[string]string)
	for key, value := range connectionAsserted {
		switch key := key.(type) {
		case string:
			switch value := value.(type) {
			case string:
				connectionString[key] = value
			}
		}
	}
	return connectionString
}

func (overlay *Overlay) SetupOpenFlowRules(rule utils.Rule, src string, dst string) error {
	shortestPath := overlay.ryuClient.FindShortestPath(src, dst)
	log.Printf("Calculated shortest path is %s\n", shortestPath)

	var dpid string
	var srcPort int
	var dstPort int
	for index, path := range shortestPath {
		if index%3 == 0 {
			srcPortInt, err := strconv.Atoi(strings.Split(path, "/")[1])
			srcPort = srcPortInt
			if err != nil {
				return errors.Wrap(err, "Failed to convert port name")
			}
		} else if index%3 == 1 {
			dpid = path
		} else if index%3 == 2 {
			dstPort, err := strconv.Atoi(strings.Split(path, "/")[1])
			if err != nil {
				return errors.Wrap(err, "Failed to convert port name")
			}
			dpidInt, err := strconv.ParseUint(dpid, 16, 64)
			if err != nil {
				return errors.Wrap(err, "Failed to convert dpid to int")
			}
			rule.Dpid = int(dpidInt)
			rule.Matching.InPort = srcPort
			rule.Action[0].Port = dstPort
			overlay.ryuClient.InstallFlow(rule)
		}
	}
	_ = srcPort
	_ = dstPort
	return nil
}

func (overlay *Overlay) UninstallOpenFlowRules(rule utils.Rule, src string, dst string) error {
	shortestPath := overlay.ryuClient.FindShortestPath(src, dst)
	log.Printf("Calculated shortest path is %s\n", shortestPath)

	var dpid string
	var srcPort int
	var dstPort int
	for index, path := range shortestPath {
		if index%3 == 0 {
			srcPortInt, err := strconv.Atoi(strings.Split(path, "/")[1])
			srcPort = srcPortInt
			if err != nil {
				return errors.Wrap(err, "Failed to convert port name")
			}
		} else if index%3 == 1 {
			dpid = path
		} else if index%3 == 2 {
			dstPort, err := strconv.Atoi(strings.Split(path, "/")[1])
			if err != nil {
				return errors.Wrap(err, "Failed to convert port name")
			}
			dpidInt, err := strconv.ParseUint(dpid, 16, 64)
			rule.Dpid = int(dpidInt)
			if err != nil {
				return errors.Wrap(err, "Failed to convert dpid to int")
			}
			rule.Matching.InPort = srcPort
			rule.Action[0].Port = dstPort
			overlay.ryuClient.DeleteFlow(rule)
		}
	}
	_ = srcPort
	_ = dstPort
	return nil
}

func ExtractSWs(overlay map[string]interface{}) map[string]Switch {
	var switches map[string]Switch

	switches = make(map[string]Switch)

	for sw, connections := range overlay {
		if _, ok := switches[sw]; !ok {
			switches[sw] = Switch{
				Name: sw,
			}
		}

		connectionsAsserted := connections.([]interface{})
		for _, connection := range connectionsAsserted {
			connectionString := FixConnectionFormat(connection)
			endpoint := connectionString["endpoint"]
			// Is it a host
			if _, ok := connectionString["ip"]; !ok {
				if _, ok := switches[endpoint]; !ok {
					switches[endpoint] = Switch{
						Name: endpoint,
					}
				}
			}
		}
	}

	return switches
}

func ExtractHosts(overlay map[string]interface{}) map[string]Host {
	var hosts map[string]Host

	hosts = make(map[string]Host)

	for _, connections := range overlay {
		connectionsAsserted := connections.([]interface{})
		for _, connection := range connectionsAsserted {
			connectionString := FixConnectionFormat(connection)
			endpoint := connectionString["endpoint"]
			// Is it a host
			if _, ok := connectionString["ip"]; ok {
				if _, ok := hosts[endpoint]; !ok {
					host := Host{}
					if _, ok := connectionString["default"]; ok {
						host = Host{
							OverlayIP: connectionString["ip"],
							Gateway:   true,
							Name:      endpoint,
						}
					} else {
						host = Host{
							OverlayIP: connectionString["ip"],
							Gateway:   false,
							Name:      endpoint,
						}
					}
					hosts[endpoint] = host
				}
			}
		}
	}

	return hosts
}

func ConfigureSwitch(osClient utils.OpenStackClient, sw Switch, vmConfiguration models.VMConfiguration) *models.VM {
	vmObj, err := models.CreateOrFindVM(&osClient, sw.Name, &vmConfiguration)
	log.Printf("Working on switch %s\n", vmObj.Name)
	if err != nil {
		log.Fatalf("VM creation failed")
	}
	vmObjD := *vmObj
	utils.InstallOVS(vmObjD)
	utils.SetOverlayInterface(vmObjD, "")
	utils.RunCommand(vmObjD, "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")
	return vmObj
}

func ConfigureHost(osClient utils.OpenStackClient, host Host, vmConfiguration models.VMConfiguration) *models.VM {
	vmObj, err := models.CreateOrFindVM(&osClient, host.Name, &vmConfiguration)
	log.Printf("Working on host %s\n", vmObj.Name)
	if err != nil {
		log.Fatalf("VM creation failed")
	}
	vmObjD := *vmObj
	utils.InstallOVS(vmObjD)
	utils.SetOverlayInterface(vmObjD, host.OverlayIP)
	utils.RunCommand(vmObjD, "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")
	if host.Gateway {
		utils.RunCommand(vmObjD, "sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE")
		utils.RunCommand(vmObjD, "sudo iptables -P FORWARD ACCEPT")
	}
	return vmObj
}

func (overlayObj *Overlay) DeployOverlay(osClient utils.OpenStackClient, overlay map[string]interface{}, vmConfiguration models.VMConfiguration, ctrlEndpoint string) {
	var switches map[string]models.VM
	var hosts map[string]models.VM
	var defaultOverlayAddress string
	var gatewayIp string

	switches = make(map[string]models.VM)
	hosts = make(map[string]models.VM)

	switchChannel := make(chan *models.VM)
	switchNames := ExtractSWs(overlay)

	hostChannel := make(chan *models.VM)
	hostNames := ExtractHosts(overlay)

	for _, sw := range switchNames {
		go func(sw Switch) {
			log.Printf("Creating VM %s\n", sw)
			switchChannel <- ConfigureSwitch(osClient, sw, vmConfiguration)
		}(sw)
	}

	for _, host := range hostNames {
		go func(host Host) {
			log.Printf("Creating VM %s\n", host)
			hostChannel <- ConfigureHost(osClient, host, vmConfiguration)
		}(host)
	}

	for range hostNames {
		host := <-hostChannel
		serviceProm := &utils.Service{
			Name:    host.Name + "-prometheus",
			Tags:    []string{"overlay", "prometheus"},
			Port:    9100,
			Address: host.IP[0],
		}

		serviceCadvisor := &utils.Service{
			Name:    host.Name + "-cadvisor",
			Tags:    []string{"overlay", "cadvisor"},
			Port:    8080,
			Address: host.IP[0],
		}
		overlayObj.consulClient.RegisterService(serviceProm)
		overlayObj.consulClient.RegisterService(serviceCadvisor)

		host.OverlayIp = hostNames[host.Name].OverlayIP
		hosts[host.Name] = *host
		if hostNames[host.Name].Gateway {
			defaultOverlayAddress = hostNames[host.Name].OverlayIP
			gatewayIp = host.IP[0]
		}
	}

	for range switchNames {
		sw := <-switchChannel
		switches[sw.Name] = *sw

		serviceProm := &utils.Service{
			Name:    sw.Name + "-prometheus",
			Tags:    []string{"overlay", "prometheus"},
			Port:    9100,
			Address: sw.IP[0],
		}

		serviceCadvisor := &utils.Service{
			Name:    sw.Name + "-cadvisor",
			Tags:    []string{"overlay", "cadvisor"},
			Port:    8080,
			Address: sw.IP[0],
		}

		overlayObj.consulClient.RegisterService(serviceProm)
		overlayObj.consulClient.RegisterService(serviceCadvisor)
	}

	for sw, connections := range overlay {
		connectionsAsserted := connections.([]interface{})
		vmSrc := switches[sw]
		for _, connection := range connectionsAsserted {
			connectionString := FixConnectionFormat(connection)

			endpoint := connectionString["endpoint"]
			var endpointVM models.VM

			// Is it a host
			if _, ok := connectionString["ip"]; ok {
				if val, ok := hosts[endpoint]; ok {
					endpointVM = val
				}
			} else {
				if val, ok := switches[endpoint]; ok {
					endpointVM = val
				}
			}
			connectNodes(vmSrc, endpointVM)
		}
	}

	log.Println("Setting the controller for switches")
	for _, sw := range switches {
		utils.RunCommand(sw, "sudo ovs-vsctl set bridge br1 protocols=OpenFlow10")
		utils.SetController(sw, ctrlEndpoint+":6633")
	}

	log.Println("Waiting for rules to be setup")
	time.Sleep(30 * time.Second)

	log.Println("Replacing default gateway with the new gateway")

	for _, vm := range hosts {
		if vm.OverlayIp != defaultOverlayAddress {
			utils.RunCommand(vm, "sshpass -p savi ssh -o StrictHostKeyChecking=no ubuntu@"+gatewayIp+" ping -c10 "+vm.OverlayIp)
			out, _ := utils.RunCommandFromOverlay(vm, "route | awk 'NR==3{print $2}'", gatewayIp)
			utils.RunCommandFromOverlay(vm, "sudo route del default gw "+string(out), gatewayIp)
			utils.RunCommandFromOverlay(vm, "sudo route add default gw "+defaultOverlayAddress, gatewayIp)
		}
	}

	// for _, sw := range switches {
	// 	cmd := "sudo ovs-ofctl show br1 | awk '/[0-9]\\(.*\\):/{print $1}' | awk -F\"[()-]\"  'BEGIN{OFS=\"-\"} {print $2,$3}'"
	// 	portString, _ := utils.RunCommand(sw, cmd)
	// 	portsTrimmed := strings.Trim(string(portString), "\n")

	// 	ports := strings.Split(portsTrimmed, "\n")

	// 	for _, port := range ports {
	// 		utils.RunCommand(sw, "sudo ovs-ofctl mod-port br1 "+port+" receive")
	// 		utils.RunCommand(sw, "sudo ovs-ofctl mod-port br1 "+port+" forward")
	// 		utils.RunCommand(sw, "sudo ovs-ofctl mod-port br1 "+port+" flood")
	// 	}
	// }
}

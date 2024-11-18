package application

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/compose-spec/compose-go/v2/types"
	api "github.com/lxc/incus/v6/shared/api"

	"log/slog"
)

// DefaultNetworkName is the stable name of the default network for a stack
func (c *Compose) DefaultNetworkName() string {
	slog.Info("DefaultNetworkName", slog.String("name", c.Name+"_network"))
	return c.Name + "_network"
}

// CreateDefaultNetwork creates the default network for a stack
func (c *Compose) CreateDefaultNetwork(nettype string) error {
	var stdinData api.NetworkPut
	if nettype == "" {
		nettype = "bridge"
	}

	// Parse remote
	resources, err := c.ParseServers(c.DefaultNetworkName())
	if err != nil {
		return err
	}

	resource := resources[0]
	client := resource.server

	// Create the network
	network := api.NetworksPost{
		NetworkPut: stdinData,
	}

	network.Name = resource.name
	network.Type = nettype

	if network.Config == nil {
		network.Config = map[string]string{}
	}

	err = client.CreateNetwork(network)
	if err != nil {
		return err
	}

	fmt.Printf("Network %s created"+"\n", resource.name)

	return nil
}

// DestroyDefaultNetwork destroys the default network for a stack
func (c *Compose) DestroyDefaultNetwork() error {

	// Parse remote
	resources, err := c.ParseServers(c.DefaultNetworkName())
	if err != nil {
		return err
	}

	resource := resources[0]

	// Delete the network
	err = resource.server.DeleteNetwork(resource.name)
	if err != nil {
		return err
	}
	fmt.Printf("Network %s deleted"+"\n", resource.name)

	return nil
}

// creates a network forward
func (c *Compose) CreateNetworkForward() error {
	// Parse remote
	resources, err := c.ParseServers(c.DefaultNetworkName())
	if err != nil {
		return err
	}

	resource := resources[0]
	// Create the network forward.

	outboundIP := GetOutboundIP()

	var forwardPut api.NetworkForwardPut
	forward := api.NetworkForwardsPost{
		ListenAddress:     outboundIP.String(),
		NetworkForwardPut: forwardPut,
	}

	forward.Normalise()

	client := resource.server

	err = client.CreateNetworkForward(resource.name, forward)
	if err != nil {
		return err
	}

	return nil
}

// remove a network forward
func (c *Compose) RemoveNetworkForward() error {
	// Parse remote
	resources, err := c.ParseServers(c.DefaultNetworkName())
	if err != nil {
		return err
	}

	resource := resources[0]
	// Create the network forward.

	outboundIP := GetOutboundIP()
	// remove the network forward
	return resource.server.DeleteNetworkForward(resource.name, outboundIP.String())
}

// add a port to a forward
func (c *Compose) CreateNetworkForwardPort(portConfig types.ServicePortConfig, targetIP string) error {
	// Parse remote
	resources, err := c.ParseServers(c.DefaultNetworkName())
	if err != nil {
		return err
	}

	resource := resources[0]
	// Create the network forward.

	outboundIP := GetOutboundIP()
	client := resource.server

	// Get the network forward.
	forward, etag, err := client.GetNetworkForward(resource.name, outboundIP.String())
	if err != nil {
		return err
	}

	port := api.NetworkForwardPort{
		Protocol:      portConfig.Protocol,
		ListenPort:    portConfig.Published,
		TargetAddress: targetIP,
	}

	port.TargetPort = strconv.FormatUint(uint64(portConfig.Target), 10)

	forward.Ports = append(forward.Ports, port)

	forward.Normalise()

	return client.UpdateNetworkForward(resource.name, forward.ListenAddress, forward.Writable(), etag)

}

// remove a port from a forward
func (c *Compose) RemoveNetworkForwardPort(types.ServicePortConfig) error {
	return nil
}

// Get preferred outbound ip of this machine
// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

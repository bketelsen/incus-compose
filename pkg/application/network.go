package application

import (
	api "github.com/lxc/incus/v6/shared/api"

	"log/slog"
)

// DefaultNetworkName is the stable name of the default network for a stack
func (c *Compose) DefaultNetworkName() string {
	slog.Info("Default Network", slog.String("name", c.Name))
	return c.Name
}

// CreateDefaultNetwork creates the default network for a stack
func (c *Compose) CreateDefaultNetwork(nettype string) error {

	// check to see if the Networks map has a default key
	// if not, return
	if _, ok := c.ComposeProject.Networks["default"]; !ok {
		return nil
	}

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

	slog.Info("Network created", "name", resource.name)

	return nil
}

// DestroyDefaultNetwork destroys the default network for a stack
func (c *Compose) DestroyDefaultNetwork() error {
	// check to see if the Networks map has a default key
	// if not, return
	if _, ok := c.ComposeProject.Networks["default"]; !ok {
		return nil
	}
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
	slog.Info("Network deleted", "name", resource.name)

	return nil
}

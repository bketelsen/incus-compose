package application

import (
	"fmt"
	"log/slog"
	"strings"

	incus "github.com/lxc/incus/v6/client"
	"gopkg.in/yaml.v3"
)

func (c *Compose) String() string {

	bb, _ := yaml.Marshal(c)
	return string(bb)
}

func (c *Compose) ListServices() []string {

	return c.ComposeProject.ServiceNames()
}
func (c *Compose) DependentsForService(s string) ([]string, error) {

	sc, err := c.ComposeProject.GetService(s)
	if err != nil {
		return nil, err
	}
	return c.ComposeProject.GetDependentsForService(sc), nil

}
func (c *Compose) StopService(s string, stateful, force bool, timeout int) error {
	return c.StopContainerForService(s, stateful, force, timeout)
}
func (c *Compose) StartService(s string, wait bool) error {
	return c.StartContainerForService(s, wait)
}
func (c *Compose) StopAll(stateful, force bool, timeout int) error {
	ss := c.ListServices()
	for _, s := range ss {
		err := c.StopContainerForService(s, stateful, force, timeout)
		if err != nil {
			if strings.Contains(err.Error(), "already stopped") {
				slog.Info("Instance already stopped", slog.String("instance", s))
			} else {
				return err
			}
		}
	}
	return nil
}
func (c *Compose) StartAll(wait bool) error {
	ss := c.ListServices()
	for _, s := range ss {
		err := c.StartContainerForService(s, wait)
		if err != nil {
			if strings.Contains(err.Error(), "already running") {
				slog.Info("Instance already running", slog.String("instance", s))
			} else {
				return err
			}
		}
	}
	return nil
}

type remoteResource struct {
	remote string
	server incus.InstanceServer
	name   string
}

func (c *Compose) ParseServers(remotes ...string) ([]remoteResource, error) {
	servers := map[string]incus.InstanceServer{}
	resources := []remoteResource{}

	for _, remote := range remotes {
		// Parse the remote
		remoteName, name, err := c.conf.ParseRemote(remote)
		if err != nil {
			return nil, err
		}

		// Setup the struct
		resource := remoteResource{
			remote: remoteName,
			name:   name,
		}

		// Look at our cache
		_, ok := servers[remoteName]
		if ok {
			resource.server = servers[remoteName]
			resources = append(resources, resource)
			continue
		}

		// New connection
		d, err := c.conf.GetInstanceServer(remoteName)
		if err != nil {
			return nil, err
		}

		resource.server = d
		servers[remoteName] = d
		resources = append(resources, resource)
	}

	return resources, nil
}

// instancesExist iterates over a list of instances (or snapshots) and checks that they exist.
func (c *Compose) instancesExist(resources []remoteResource) error {
	for _, resource := range resources {
		resource.server.UseProject(c.GetProject())
		_, _, err := resource.server.GetInstance(resource.name)
		if err != nil {
			return fmt.Errorf("Failed checking instance exists \"%s:%s\": %w", resource.remote, resource.name, err)
		}
	}

	return nil
}

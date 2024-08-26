package client

import (
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

type IncusClient struct {
	client incus.InstanceServer
}

func NewIncusClient() (*IncusClient, error) {
	c, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket", nil)
	if err != nil {
		c, err = incus.ConnectIncusUnix("/run/incus/unix.socket", nil)
		if err != nil {
			return nil, err
		}
	}

	return &IncusClient{client: c}, nil
}

func (i *IncusClient) WithProject(project string) {

	i.client = i.client.UseProject(project)
}

func (i *IncusClient) GetProjectNames() ([]string, error) {

	return i.client.GetProjectNames()
}

func (i *IncusClient) GetProfileNames() ([]string, error) {

	return i.client.GetProfileNames()
}

func (i *IncusClient) GetStoragePoolNames() ([]string, error) {

	return i.client.GetStoragePoolNames()
}

func (i *IncusClient) GetInstance(name string) (*api.Instance, string, error) {

	return i.client.GetInstance(name)
}
func (i *IncusClient) GetInstanceState(name string) (*api.InstanceState, string, error) {

	return i.client.GetInstanceState(name)
}

func (i *IncusClient) StartInstance(name string) error {

	action := "start"
	state := false
	if action == "start" {
		current, _, err := i.client.GetInstance(name)
		if err != nil {
			return err
		}

		// "start" for a frozen instance means "unfreeze"
		if current.StatusCode == api.Frozen {
			action = "unfreeze"
		}

		// Always restore state (if present) unless asked not to
		if action == "start" && current.Stateful {
			state = true
		}
	}

	req := api.InstanceStatePut{
		Action:  action,
		Timeout: 10,

		Stateful: state,
	}

	op, err := i.client.UpdateInstanceState(name, req, "")
	if err != nil {
		return err
	}
	return op.Wait()

}

func (i *IncusClient) CreateProfile(name string, data api.ProfilePut) error {
	// Create the profile
	profile := api.ProfilesPost{}
	profile.Name = name
	profile.ProfilePut = data

	err := i.client.CreateProfile(profile)
	if err != nil {
		return err
	}

	return nil

}
func (i *IncusClient) DeleteProfile(name string) error {
	return i.client.DeleteProfile(name)
}

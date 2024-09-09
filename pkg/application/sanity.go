package application

import (
	"fmt"
	"slices"

	incus "github.com/lxc/incus/v6/client"
)

type SanityCheckError struct {
	Step string
	Err  error
}

func (e *SanityCheckError) Error() string { return "Sanity Check: " + e.Step + " -> " + e.Err.Error() }

func (app *Compose) SanityCheck() error {
	var err error
	var remote string
	var d incus.InstanceServer
	var projectNames []string
	var profileNames []string
	var poolNames []string
	var netNames []string

	// check to see if the incus connection is valid
	// get the first service and try to connect to the incus remote
	for _, service := range app.Services {
		remote, _, err = app.conf.ParseRemote(service.Name)
		if err != nil {
			return &SanityCheckError{
				Step: "parse incus remote",
				Err:  fmt.Errorf("error parsing remote: %s", err),
			}
		}

		d, err = app.conf.GetInstanceServer(remote)
		if err != nil {
			return &SanityCheckError{
				Step: "get incus remote",
				Err:  fmt.Errorf("error getting instance server: %s", err),
			}

		}
		// only get project names once
		if len(projectNames) == 0 {
			// get the project names while we're connected
			projectNames, err = d.GetProjectNames()
			if err != nil {
				return &SanityCheckError{
					Step: "get project names",
					Err:  fmt.Errorf("error getting project names: %s", err),
				}
			}
		}
	}
	// check to see if the project exists
	if !slices.Contains(projectNames, app.GetProject()) {
		return &SanityCheckError{
			Step: "check declared project exists",
			Err:  fmt.Errorf("project '%s' does not exist", app.GetProject()),
		}
	}
	// check to see if the profiles exist
	d.UseProject(app.GetProject())
	profileNames, err = d.GetProfileNames()
	if err != nil {
		return &SanityCheckError{
			Step: "get profile names",
			Err:  fmt.Errorf("error getting profile names: %s", err),
		}
	}
	poolNames, err = d.GetStoragePoolNames()
	if err != nil {
		return &SanityCheckError{
			Step: "get storage pool names",
			Err:  fmt.Errorf("error getting storage pool names: %s", err),
		}
	}
	netNames, err = d.GetNetworkNames()
	if err != nil {
		return &SanityCheckError{
			Step: "get network names",
			Err:  fmt.Errorf("error getting network names: %s", err),
		}
	}
	// check to see if the default profiles exists
	for _, p := range app.Profiles {
		if !slices.Contains(profileNames, p) {
			return &SanityCheckError{
				Step: "check declared profile exists",
				Err:  fmt.Errorf("profile '%s' does not exist in project '%s'", p, app.GetProject()),
			}
		}
	}
	// check to see if the additional profiles exists
	for _, s := range app.Services {
		for _, p := range s.AdditionalProfiles {
			if !slices.Contains(profileNames, p) {
				return &SanityCheckError{
					Step: "check declared profile exists",
					Err:  fmt.Errorf("additional profile '%s' does not exist in project '%s'", p, app.GetProject()),
				}
			}
		}
	}
	// check to see if the instance declared storage pool exists
	for _, s := range app.Services {
		if s.Storage != "" {
			if !slices.Contains(poolNames, s.Storage) {
				return &SanityCheckError{
					Step: "check declared storage pool exists",
					Err:  fmt.Errorf("storage pool '%s' does not exist in project '%s'", s.Storage, app.GetProject()),
				}
			}
		}

		// check to see if the volume declared storage pool exists
		for _, v := range s.Volumes {
			if v.Pool != "" {
				if !slices.Contains(poolNames, v.Pool) {
					return &SanityCheckError{
						Step: "check declared volume storage pool exists",
						Err:  fmt.Errorf("volume %s: storage pool '%s' does not exist in project '%s'", v.Name, v.Pool, app.GetProject()),
					}
				}
			}
		}
	}
	// check to see if the network declared exists
	for _, s := range app.ComposeProject.Services {
		if s.Networks != nil {
			for name := range s.Networks {
				if name == "default" {
					continue
				}
				if !slices.Contains(netNames, name) {
					return &SanityCheckError{
						Step: "check declared network exists",
						Err:  fmt.Errorf("network '%s' does not exist in project '%s'", name, app.GetProject()),
					}
				}
			}
		}
	}

	return nil
}

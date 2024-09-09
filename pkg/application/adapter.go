// adapted from https://github.com/DefangLabs/defang
// LICENSE MIT License
// Copyright (c) 2024 Defang Software Labs
package application

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/gosimple/slug"
	cliconfig "github.com/lxc/incus/v6/shared/cliconfig"
)

func BuildDirect(p *types.Project, conf *cliconfig.Config) (*Compose, error) {
	compose := &Compose{}
	compose.ComposeProject = p
	compose.Name = p.Name
	compose.Project = "default"
	compose.conf = conf

	// parse extensions
	for k, v := range p.Extensions {
		switch k {
		case "x-incus-default-profiles":
			list, ok := v.([]interface{})
			if ok {
				for _, profile := range list {
					p := profile.(string)
					compose.Profiles = append(compose.Profiles, p)
				}
			}
			continue
		case "x-incus-project":
			//fmt.Printf("project %q: extension: %q value: %q\n", p.Name, k, v)
			proj, ok := v.(string)
			if ok {
				compose.Project = proj
			}
			continue
		default:
			fmt.Printf("project %q: unsupported compose extension: %q\n", p.Name, k)
		}
	}

	// parse services
	compose.Services = make(map[string]Service)
	for _, s := range p.Services {
		service := parseService(s)
		compose.Services[s.Name] = service
	}

	// get additional information about volumes
	for _, vol := range p.Volumes {
		//fmt.Println(vol.Name)
		pool := "default"
		driverPool := vol.DriverOpts["pool"]
		if driverPool != "" {
			pool = driverPool
		}
		var snap *Snapshot

		// parse volume extensions
		for k, v := range vol.Extensions {
			switch k {
			case "x-incus-snapshot":
				snapshot, ok := v.(map[string]interface{})
				if ok {
					snap = &Snapshot{}
					for k, v := range snapshot {
						switch k {
						case "schedule":
							snap.Schedule = v.(string)
						case "expiry":
							snap.Expiry = v.(string)
						case "pattern":
							snap.Pattern = v.(string)
						default:
							fmt.Printf("service %q: unsupported snapshot extension: %q\n", vol.Name, k)
						}
					}
					//service.Snapshot = snap

				}
				continue
			default:
				fmt.Printf("volume %q: unsupported compose extension: %q\n", vol.Name, k)
			}
		}
		// now find the service that uses this volume
		for _, s := range compose.Services {
			for k, v := range s.Volumes {
				fullVolName := p.Name + "_" + k
				if fullVolName == vol.Name {
					v.Pool = pool
					v.Snapshot = snap
					v.Name = v.CreateName(p.Name, s.Name, k)
				}
			}
		}
	}
	return compose, nil
}
func parseService(s types.ServiceConfig) Service {
	service := Service{}
	for dep := range s.DependsOn {
		service.DependsOn = append(service.DependsOn, dep)
	}
	service.Name = s.Name
	service.Environment = make(map[string]*string)
	for k, v := range s.Environment {
		service.Environment[k] = v
	}

	// parse service extensions
	for k, v := range s.Extensions {
		switch k {
		case "x-incus-additional-profiles":
			list, ok := v.([]interface{})
			if ok {
				for _, profile := range list {
					p := profile.(string)
					service.AdditionalProfiles = append(service.AdditionalProfiles, p)
				}
			}
			continue
		case "x-incus-cloud-init-user-data-file":
			df, ok := v.(string)
			if ok {
				service.CloudInitUserDataFile = df
			}
			continue
		case "x-incus-storage":
			pool, ok := v.(string)
			if ok {
				service.Storage = pool
			}
			continue
		case "x-incus-gpu":
			gpu, ok := v.(bool)
			if ok {
				service.GPU = gpu
			}
			continue
		case "x-incus-snapshot":
			snapshot, ok := v.(map[string]interface{})
			if ok {
				//fmt.Println("parsed snapshot", snapshot)
				snap := &Snapshot{}
				for k, v := range snapshot {
					switch k {
					case "schedule":
						snap.Schedule = v.(string)
					case "expiry":
						snap.Expiry = v.(string)
					case "pattern":
						snap.Pattern = v.(string)
					default:
						fmt.Printf("service %q: unsupported snapshot extension: %q\n", s.Name, k)
					}
				}
				service.Snapshot = snap

			}
			continue
		default:
			fmt.Printf("service %q: unsupported compose extension: %q\n", s.Name, k)
		}
	}
	service.Volumes = make(map[string]*Volume)
	service.BindMounts = make(map[string]Bind)

	// parse volumes
	for _, v := range s.Volumes {
		//volume := Volume{}
		//fmt.Println("volume type", v.Type)
		switch v.Type {
		case "volume":
			volume := &Volume{}
			volume.Mountpoint = v.Target
			service.Volumes[v.Source] = volume
		case "bind":
			bind := Bind{}
			bind.Source = v.Source
			bind.Target = v.Target
			bind.Type = "disk"
			for key, val := range v.Extensions {
				switch key {
				case "x-incus-shift":
					bind.Shift = val.(bool)
					continue
				default:
					fmt.Printf("volume %q: unsupported compose extension: %q\n", v, key)
				}
			}
			service.BindMounts[bindNameStable(v.Source)] = bind
		default:
			fmt.Printf("service %q: unsupported volume type: %q\n", s.Name, v.Type)
		}

	}

	service.Image = s.Image
	if s.ContainerName != "" {
		service.ContainerName = s.ContainerName
	}

	return service

}

func bindNameStable(path string) string {
	return slug.Make(path)
}

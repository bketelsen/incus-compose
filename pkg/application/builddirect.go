// adapted from https://github.com/DefangLabs/defang
// LICENSE MIT License
// Copyright (c) 2024 Defang Software Labs
package application

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

func BuildDirect(p *types.Project) (*Compose, error) {
	compose := &Compose{}
	compose.ComposeProject = p
	compose.Name = p.Name
	compose.Project = "default"

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
					//fmt.Println("parsed snapshot", snapshot)
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
				//fmt.Println("k, name", k, fullVolName)
				if fullVolName == vol.Name {
					v.Pool = pool
					v.Snapshot = snap
				}
			}
		}
	}
	return compose, nil
}

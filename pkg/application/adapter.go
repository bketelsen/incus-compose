// adapted from https://github.com/DefangLabs/defang
// LICENSE MIT License
// Copyright (c) 2024 Defang Software Labs
package application

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"

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
			proj, ok := v.(string)
			if ok {
				compose.Project = proj
			}
			continue
		default:
			slog.Error("unsupported compose extension", "project", p.Name, "extension", k)
		}
	}

	// parse services
	compose.Services = make(map[string]Service)
	for _, s := range p.Services {
		service := parseService(s)
		compose.Services[s.Name] = service
	}

	// parse secretsfiles
	compose.SecretsFiles = make(map[string]SecretsFile)
	for _, s := range p.Secrets {
		sf := parseSecret(s)
		compose.SecretsFiles[s.Name] = sf
	}

	// get additional information about volumes
	for _, vol := range p.Volumes {
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
							slog.Error("unsupported snapshot extension", "volume", vol.Name, "extension", k)
						}
					}
					//service.Snapshot = snap

				}
				continue
			default:
				slog.Error("unsupported compose extension", "volume", vol.Name, "extension", k)
			}
		}
		// now find the service that uses this volume
		for _, s := range compose.Services {
			for k, v := range s.Volumes {
				fullVolName := p.Name + "_" + k
				if fullVolName == vol.Name {
					if pool, exists := vol.DriverOpts["pool"]; exists && pool != "" {
						v.Pool = pool
					} else if s.Storage != "" {
						v.Pool = s.Storage
					} else {
						v.Pool = "default"
					}
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
						slog.Error("unsupported snapshot extension", "service", s.Name, "extension", k)
					}
				}
				service.Snapshot = snap

			}
			continue
		default:
			slog.Error("unsupported compose extension", "service", s.Name, "extension", k)
		}
	}
	service.Volumes = make(map[string]*Volume)
	service.BindMounts = make(map[string]Bind)

	// parse volumes
	for _, v := range s.Volumes {

		var shifted bool = false
		for key, val := range v.Extensions {
			switch key {
			case "x-incus-shift":
				shifted = val.(bool)
				continue
			default:
				slog.Error("unsupported compose extension", "volume", v.Source, "extension", key)
			}
		}

		switch v.Type {
		case "volume":
			volume := &Volume{}
			volume.Mountpoint = v.Target
			volume.ReadOnly = v.ReadOnly
			if shifted {
				volume.Shift = shifted
			}
			service.Volumes[v.Source] = volume
		case "bind":
			bind := Bind{}
			bind.Source = v.Source
			bind.Target = v.Target
			bind.Type = "disk"
			bind.ReadOnly = v.ReadOnly
			if shifted {
				bind.Shift = shifted
			}

			for key, val := range v.Extensions {
				switch key {
				case "x-incus-shift":
					bind.Shift = val.(bool)
					continue
				default:
					slog.Error("unsupported compose extension", "volume", v.Source, "extension", key)
				}
			}
			service.BindMounts[bindNameStable(v.Source)] = bind
		default:
			slog.Error("unsupported volume type", "service", s.Name, "volume", v.Source, "type", v.Type)
		}

	}

	service.Secrets = make(map[string]Secret)
	for _, v := range s.Secrets {
		s := Secret{}
		s.MountPoint = v.Target
		service.Secrets[v.Source] = s
	}

	service.Image = s.Image
	if s.ContainerName != "" {
		service.ContainerName = s.ContainerName
	}

	return service

}

func bindNameStable(path string) string {
	name := slug.Make(path)
	if len(name) > 64 {
		sha256sum := sha256.Sum256([]byte(name))
		name = hex.EncodeToString(sha256sum[:16])
	}

	return name
}

func parseSecret(s types.SecretConfig) SecretsFile {
	sf := SecretsFile{}
	sf.FilePath = s.File
	return sf
}

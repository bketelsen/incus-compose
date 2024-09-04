package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	cli "github.com/bketelsen/incus-compose/pkg/incus"
	incus "github.com/lxc/incus/v6/client"

	"github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) RemoveContainerForService(service string, force bool) error {
	slog.Info("Removing", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	return app.removeInstance(service, force)

}
func (app *Compose) StopContainerForService(service string, stateful, force bool, timeout int) error {
	slog.Info("Stopping", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	return app.updateInstanceState(service, "stop", timeout, force, stateful)

}
func (app *Compose) StartContainerForService(service string, wait bool) error {
	slog.Info("Starting", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	d, err := app.getInstanceServer(service)
	if err != nil {
		return err
	}

	inst, _, _ := d.GetInstance(service)
	if inst != nil && inst.Name == service && inst.Status == "Running" {
		slog.Info("Instance already running", slog.String("instance", service))
	} else {
		err = app.updateInstanceState(service, "start", -1, false, false)
		if err != nil {
			return err
		}
	}

	if wait {
		if svc.CloudInitUserData != "" || svc.CloudInitUserDataFile != "" {
			slog.Info("cloud-init", slog.String("instance", service), slog.String("status", "waiting"))

			args := []string{"exec", service}
			args = append(args, "--project", app.GetProject())
			args = append(args, "--", "cloud-init", "status", "--wait")
			out, code, err := cli.ExecuteShellStreamExitCode(context.Background(), args)
			if err != nil {
				slog.Error("Incus error", slog.String("instance", service), slog.String("message", out))
				return err
			}
			if code == 2 {
				slog.Error("cloud-init", slog.String("instance", service), slog.String("status", "completed with recoverable errors"))
			}

			slog.Info("cloud-init", slog.String("instance", service), slog.String("status", "done"))
			slog.Debug("Incus ", slog.String("instance", service), slog.String("message", out))
		}
	}
	return nil
}

func (app *Compose) RestartContainerForService(service string) error {
	slog.Info("Restarting", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	return app.updateInstanceState(service, "restart", -1, false, false)

}
func (app *Compose) InitContainerForService(service string) error {
	slog.Info("Initialize", slog.String("instance", service))

	var image string
	var iremote string

	sc, err := app.ComposeProject.GetService(service)
	if err != nil {
		return err
	}

	// Parse the remote
	remote, name, err := app.conf.ParseRemote(service)
	if err != nil {
		return err
	}

	d, err := app.conf.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	inst, _, _ := d.GetInstance(service)
	if inst != nil && inst.Name == service {
		slog.Info("Instance found", slog.String("instance", service))
		return nil
	}

	var instancePost api.InstancesPost
	var devicesMap map[string]map[string]string
	var configMap map[string]string
	var profiles []string
	var userDataFile string
	var storageOverride string
	var assignGPU bool
	var instanceSnapshot *Snapshot

	// add the profiles specified in the compose file
	profiles = append(profiles, app.GetProfiles()...)

	// parse service extensions
	for k, v := range sc.Extensions {
		switch k {
		case "x-incus-additional-profiles":
			list, ok := v.([]interface{})
			if ok {
				for _, profile := range list {
					p := profile.(string)
					profiles = append(profiles, p)
				}
			}
			continue
		case "x-incus-cloud-init-user-data-file":
			df, ok := v.(string)
			if ok {
				// save it to use later
				userDataFile = df
			}
			continue
		case "x-incus-storage":
			pool, ok := v.(string)
			if ok {
				// save it to use later
				storageOverride = pool
			}
			continue
		case "x-incus-gpu":
			gpu, ok := v.(bool)
			if ok {
				// save it to use later
				assignGPU = gpu
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
						fmt.Printf("service %q: unsupported snapshot configuration: %q\n", sc.Name, k)
					}
				}
				instanceSnapshot = snap

			}
			continue
		default:
			fmt.Printf("service %q: unsupported compose extension: %q\n", sc.Name, k)
		}
	}

	// set up deviceMap
	devicesMap = map[string]map[string]string{}
	if len(sc.Networks) > 0 {
		networkNumber := 0
		for net := range sc.Networks {
			if net != "default" {
				netName := fmt.Sprintf("eth%d", networkNumber)

				network, _, err := d.GetNetwork(net)
				if err != nil {
					return fmt.Errorf("failed loading network %q: %w", net, err)
				}

				// Prepare the instance's NIC device entry.
				var device map[string]string

				if network.Managed && d.HasExtension("instance_nic_network") {
					// If network is snapmanaged, use the network property rather than nictype, so that the
					// network's inherited properties are loaded into the NIC when started.
					device = map[string]string{
						"name":    netName,
						"type":    "nic",
						"network": network.Name,
					}
				} else {
					// If network is unmanaged default to using a macvlan connected to the specified interface.
					device = map[string]string{
						"name":    netName,
						"type":    "nic",
						"nictype": "macvlan",
						"parent":  net,
					}

					if network.Type == "bridge" {
						// If the network type is an unmanaged bridge, use bridged NIC type.
						device["nictype"] = "bridged"
					}
				}
				devicesMap[netName] = device
				networkNumber++
			}
		}
	} // sc.networks

	// config
	configMap = map[string]string{}
	for k, v := range sc.Environment {
		configMap["environment."+k] = *v
	}

	// add env vars from file
	if len(sc.EnvFiles) > 0 {
		for _, value := range sc.EnvFiles {
			r, err := readEnvironmentFile(value.Path)
			if err != nil {
				return fmt.Errorf("failed reading env file %s: %w", value.Path, err)
			}
			for k, v := range r {
				configMap["environment."+k] = v
			}
		}
	}

	// overridden storage
	if storageOverride != "" {
		_, _, err := d.GetStoragePool(storageOverride)
		if err != nil {
			return fmt.Errorf("failed loading storage pool %q: %w", storageOverride, err)
		}

		devicesMap["root"] = map[string]string{
			"type": "disk",
			"path": "/",
			"pool": storageOverride,
		}
	}

	instancePost.Name = sc.Name
	instancePost.Type = api.InstanceTypeContainer
	instancePost.InstanceType = "" // c2.micro etc
	instancePost.Config = configMap
	instancePost.Ephemeral = false
	instancePost.Description = app.Name + "-" + sc.Name
	instancePost.Profiles = profiles

	// gpu
	if assignGPU {
		devicesMap[sc.Name+"GPU"] = map[string]string{
			"type": "gpu",
		}
	}
	if instanceSnapshot != nil {
		configMap["snapshots.schedule"] = instanceSnapshot.Schedule
		configMap["snapshots.pattern"] = instanceSnapshot.Pattern
		configMap["snapshots.expiry"] = instanceSnapshot.Expiry

	}

	if userDataFile != "" {
		bb, err := os.ReadFile(userDataFile)
		if err != nil {
			slog.Error("Loading cloud-init", slog.String("error", err.Error()))
			return err
		}
		configMap["user.user-data"] = string(bb)
	}

	instancePost.Devices = devicesMap

	iremote, image, err = app.conf.ParseRemote(sc.Image)
	if err != nil {
		return err
	}

	iremote, image = guessImage(app.conf, d, remote, iremote, image)
	// Deal with the default image
	if image == "" {
		image = "default"
	}
	imgRemote, imgInfo, err := getImgInfo(d, app.conf, iremote, remote, image, &instancePost.Source)
	if err != nil {
		return err
	}

	if app.conf.Remotes[iremote].Protocol == "incus" {

		instancePost.Type = api.InstanceType(imgInfo.Type)
	}
	fmt.Println(instancePost)

	op, err := d.CreateInstanceFromImage(imgRemote, *imgInfo, instancePost)
	if err != nil {
		return err
	}
	err = op.Wait()
	if err != nil {
		return err
	}
	fmt.Println(name)

	return nil

}

func (app *Compose) updateInstanceState(name string, state string, timeout int, force bool, stateful bool) error {
	remote, name, err := app.conf.ParseRemote(name)
	if err != nil {
		return err
	}

	d, err := app.conf.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	req := api.InstanceStatePut{
		Action:   state,
		Timeout:  timeout,
		Force:    force,
		Stateful: stateful,
	}

	op, err := d.UpdateInstanceState(name, req, "")
	if err != nil {
		return err
	}

	return op.Wait()
}

func (app *Compose) getInstanceServer(name string) (incus.InstanceServer, error) {
	remote, _, err := app.conf.ParseRemote(name)
	if err != nil {
		return nil, err
	}

	return app.conf.GetInstanceServer(remote)

}
func (app *Compose) removeInstance(name string, force bool) error {

	// Parse remote
	resources, err := app.ParseServers(name)
	if err != nil {
		return err
	}

	// Check that everything exists.
	err = instancesExist(resources)
	if err != nil {
		return err
	}

	// Process with deletion.
	for _, resource := range resources {
		connInfo, err := resource.server.GetConnectionInfo()
		if err != nil {
			return err
		}

		ct, _, err := resource.server.GetInstance(resource.name)
		if err != nil {
			return err
		}

		if ct.StatusCode != 0 && ct.StatusCode != api.Stopped {
			if !force {
				return fmt.Errorf("The instance is currently running, stop it first or pass --force")
			}

			req := api.InstanceStatePut{
				Action:  "stop",
				Timeout: -1,
				Force:   true,
			}

			op, err := resource.server.UpdateInstanceState(resource.name, req, "")
			if err != nil {
				return err
			}

			err = op.Wait()
			if err != nil {
				return fmt.Errorf("Stopping the instance failed: %s", err)
			}

			if ct.Ephemeral {
				continue
			}
		}

		// if c.flagForceProtected && util.IsTrue(ct.ExpandedConfig["security.protection.delete"]) {
		// 	// Refresh in case we had to stop it above.
		// 	ct, etag, err := resource.server.GetInstance(resource.name)
		// 	if err != nil {
		// 		return err
		// 	}

		// 	ct.Config["security.protection.delete"] = "false"
		// 	op, err := resource.server.UpdateInstance(resource.name, ct.Writable(), etag)
		// 	if err != nil {
		// 		return err
		// 	}

		// 	err = op.Wait()
		// 	if err != nil {
		// 		return err
		// 	}
		// }

		// Instance delete
		op, err := resource.server.DeleteInstance(name)
		if err != nil {
			return fmt.Errorf("Failed deleting instance %q in project %q: %w", resource.name, connInfo.Project, err)
		}

		return op.Wait()
	}
	return nil

}

func readEnvironmentFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Split the file into lines.
	lines := strings.Split(string(content), "\n")

	// Create a map to store the key value pairs.
	envMap := make(map[string]string)

	// Iterate over the lines.
	for _, line := range lines {
		if line == "" {
			continue
		}

		pieces := strings.SplitN(line, "=", 2)
		value := ""
		if len(pieces) > 1 {
			value = pieces[1]
		}

		envMap[pieces[0]] = value
	}

	return envMap, nil
}

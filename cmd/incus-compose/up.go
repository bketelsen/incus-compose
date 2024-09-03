/*
Copyright © 2024 Brian Ketelsen <bketelsen@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"

	"github.com/bketelsen/incus-compose/internal/i18n"
	"github.com/bketelsen/incus-compose/pkg/application"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/spf13/cobra"
)

type cmdUp struct {
	global       *cmdGlobal
	action       *cmdAction
	volumeCreate *cmdStorageVolumeCreate
	volumeShow   *cmdStorageVolumeShow
}

// downCmd represents the down command
func (c *cmdUp) Command() *cobra.Command {
	cmdAction := cmdAction{global: c.global}
	c.action = &cmdAction

	c.volumeCreate = &cmdStorageVolumeCreate{global: c.global}
	c.volumeShow = &cmdStorageVolumeShow{global: c.global}

	cmd := &cobra.Command{}
	cmd.Use = "up"
	cmd.Short = "Create and start instances"
	cmd.Long = `Create and start instances`
	cmd.RunE = c.Run

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return c.global.cmpInstances(toComplete)
	}

	return cmd
}

// downCmd represents the down command
func (c *cmdUp) Run(cmd *cobra.Command, args []string) error {

	for _, service := range c.global.compose.Order(true) {

		err := c.InitContainerForService(service)
		if err != nil {
			return err
		}

		err = c.CreateVolumesForService(service)
		if err != nil {
			return err
		}

		err = c.CreateBindsForService(service)
		if err != nil {
			return err
		}

		err = c.AttachVolumesForService(service)
		if err != nil {
			return err
		}

		err = c.StartContainerForService(service)
		if err != nil {
			return err
		}

	}
	return nil
}
func (c *cmdUp) StartContainerForService(service string) error {
	err := c.action.doAction("start", c.global.conf, service)
	if err != nil {
		return err
	}
	if !c.global.flagQuiet {
		fmt.Printf(i18n.G("Instance %s started")+"\n", service)
	}
	return nil

}
func (c *cmdUp) AttachVolumesForService(service string) error {
	svc, ok := c.global.compose.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for volName, vol := range svc.Volumes {
		err := c.attachVolume(vol.Name(c.global.compose.Name, service, volName), service, vol)
		if err != nil {
			return err
		}
	}

	return nil

}
func (c *cmdUp) attachVolume(volName, instance string, vol *application.Volume) error {
	remote, name, err := c.global.conf.ParseRemote(volName)
	if err != nil {
		return err
	}

	d, err := c.global.conf.GetInstanceServer(remote)
	if err != nil {
		return err
	}
	inst, etag, err := d.GetInstance(instance)
	if err != nil {
		return err
	}

	// Check if device exists
	_, ok := inst.Devices[name]
	if ok {
		slog.Info("Device already exists", slog.String("volume", name))
		return nil
	}

	volName, volType := parseVolume("custom", volName)
	if volType != "custom" {
		return fmt.Errorf("Only \"custom\" volumes can be attached to instances")
	}

	// Prepare the instance's device entry
	dev := map[string]string{
		"type":   "disk",
		"pool":   vol.Pool,
		"source": volName,
		"path":   vol.Mountpoint,
	}

	inst.Devices[name] = dev

	op, err := d.UpdateInstance(instance, inst.Writable(), etag)
	if err != nil {
		return err
	}

	return op.Wait()

}
func (c *cmdUp) CreateBindsForService(service string) error {
	svc, ok := c.global.compose.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for bindName, bind := range svc.BindMounts {

		device := map[string]string{}
		device["type"] = bind.Type
		device["source"] = bind.Source
		device["path"] = bind.Target
		if bind.Shift {
			device["shift"] = "true"
		}
		remote, name, err := c.global.conf.ParseRemote(service)
		if err != nil {
			return err
		}

		d, err := c.global.conf.GetInstanceServer(remote)
		if err != nil {
			return err
		}

		inst, etag, err := d.GetInstance(name)
		if err != nil {
			return err
		}

		_, ok := inst.Devices[bindName]
		if ok {
			slog.Info("Device already exists", slog.String("name", bindName))
			return nil
		}

		inst.Devices[name] = device

		op, err := d.UpdateInstance(name, inst.Writable(), etag)
		if err != nil {
			return err
		}

		err = op.Wait()
		if err != nil {
			return err
		}
		if !c.global.flagQuiet {
			fmt.Printf(i18n.G("Binding %s to %s on instance %s")+"\n", bind.Source, bind.Target, service)
		}

		return nil
	}
	return nil
}

func (c *cmdUp) CreateVolumesForService(service string) error {

	svc, ok := c.global.compose.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for volName, vol := range svc.Volumes {

		completeName := vol.Name(c.global.compose.Name, service, volName)
		slog.Debug("Volume", slog.String("name", completeName), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		existingVolume, _ := c.getVolume(vol.Pool, completeName)

		if existingVolume != nil && completeName == existingVolume.Name {
			slog.Info("Volume found", slog.String("volume", completeName))
		} else {
			err := c.createVolume(completeName, *vol, *vol.Snapshot)
			if err != nil {
				return err
			}
			if !c.global.flagQuiet {
				fmt.Printf(i18n.G("Storage volume %s created")+"\n", completeName)
			}
		}
	}

	return nil
}

func (c *cmdUp) createVolume(name string, vol application.Volume, snapshot application.Snapshot) error {

	snapargs := make(map[string]string)

	if snapshot.Schedule != "" {
		snapargs["snapshots.schedule"] = snapshot.Schedule
	}
	if snapshot.Pattern != "" {
		snapargs["snapshots.pattern"] = snapshot.Pattern
	}
	if snapshot.Expiry != "" {
		snapargs["snapshots.expiry"] = snapshot.Expiry
	}

	volName, volType := parseVolume("custom", name)

	var volumePut api.StorageVolumePut

	// Create the storage volume entry
	volume := api.StorageVolumesPost{
		Name:             volName,
		Type:             volType,
		ContentType:      "filesystem",
		StorageVolumePut: volumePut,
	}

	if volumePut.Config == nil {
		volume.Config = map[string]string{}
	}

	for k, v := range snapargs {
		volume.Config[k] = v
	}
	// Parse remote
	resources, err := c.global.ParseServers(vol.Pool)
	if err != nil {
		return err
	}

	resource := resources[0]

	if resource.name == "" {
		return fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	err = client.CreateStoragePoolVolume(vol.Pool, volume)
	if err != nil {
		return err
	}

	return nil

}

func (c *cmdUp) getVolume(pool, volume string) (*api.StorageVolume, error) {

	// Parse remote
	resources, err := c.global.ParseServers(pool)
	if err != nil {
		return nil, err
	}

	resource := resources[0]

	if resource.name == "" {
		return nil, fmt.Errorf(i18n.G("Missing pool name"))
	}

	client := resource.server

	// Parse the input
	volName, volType := parseVolume("custom", volume)

	// If a target member was specified, get the volume with the matching
	// name on that member, if any.
	// if c.storage.flagTarget != "" {
	// 	client = client.UseTarget(c.storage.flagTarget)
	// }

	// Get the storage volume entry
	vol, _, err := client.GetStoragePoolVolume(resource.name, volType, volName)
	if err != nil {
		// Give more context on missing volumes.
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			if volType == "custom" {
				return nil, fmt.Errorf("Storage pool volume \"%s/%s\" not found. Try virtual-machine or container for type", volType, volName)
			}

			return nil, fmt.Errorf("Storage pool volume \"%s/%s\" not found", volType, volName)
		}

		return nil, err
	}

	sort.Strings(vol.UsedBy)

	return vol, nil

}

func (c *cmdUp) InitContainerForService(service string) error {
	fmt.Println("InitContainerForService")
	var image string
	var remote string
	var iremote string

	sc, err := c.global.compose.ComposeProject.GetService(service)
	if err != nil {
		return err
	}
	// client, err := client.NewIncusClient()
	// if err != nil {
	// 	return err
	// }
	// client.WithProject(c.global.compose.GetProject())

	iremote, image, err = c.global.conf.ParseRemote(sc.Image)
	if err != nil {
		return err
	}
	remote, _, err = c.global.conf.ParseRemote(sc.Name)
	if err != nil {
		return err
	}

	d, err := c.global.conf.GetInstanceServer(remote)
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
	var instanceSnapshot *application.Snapshot

	// add the profiles specified in the compose file
	profiles = append(profiles, c.global.compose.GetProfiles()...)

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
				snap := &application.Snapshot{}
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
			if net == "default" {
				continue
			}
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
	} // sc.networks

	// config
	configMap = map[string]string{}
	for k, v := range sc.Environment {
		configMap["environment."+k] = *v
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
	instancePost.Description = c.global.compose.Name + "-" + sc.Name
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
	iremote, image = guessImage(c.global.conf, d, remote, iremote, image)
	// Deal with the default image
	if image == "" {
		image = "default"
	}
	imgRemote, imgInfo, err := getImgInfo(d, c.global.conf, iremote, remote, image, &instancePost.Source)
	if err != nil {
		return err
	}
	if c.global.conf.Remotes[iremote].Protocol == "incus" {

		instancePost.Type = api.InstanceType(imgInfo.Type)
	}

	op, err := d.CreateInstanceFromImage(imgRemote, *imgInfo, instancePost)
	if err != nil {
		return err
	}
	err = op.Wait()
	if err != nil {
		return err
	}
	if !c.global.flagQuiet {
		fmt.Printf(i18n.G("Instance %s created")+"\n", sc.Name)
	}

	return nil

}

package application

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	config "github.com/lxc/incus/v6/shared/cliconfig"
)

func (app *Compose) CreateCloudProfileForService(service string) (string, error) {

	svc, ok := app.Services[service]
	if !ok {
		return "", fmt.Errorf("service %s not found", service)
	}
	name, err := app.ProfileNameForService(service)
	if err != nil {
		return "", err
	}
	put := api.ProfilePut{}
	put.Description = "Cloud profile for " + app.Project + "/" + service

	needsProfile := false

	if svc.CloudInitUserData != "" {
		put.Config = map[string]string{"user.user-data": svc.CloudInitUserData}
		needsProfile = true
	}
	if svc.CloudInitUserDataFile != "" {
		bb, err := os.ReadFile(svc.CloudInitUserDataFile)
		if err != nil {
			slog.Error("Loading cloud-init", slog.String("error", err.Error()))
			return "", err
		}
		put.Config = map[string]string{"user.user-data": string(bb)}
		needsProfile = true
	}

	if needsProfile {
		slog.Info("Creating custom cloud-init profile", slog.String("instance", service), slog.String("profile", name))

		client, err := client.NewIncusClient()
		if err != nil {
			return "", err
		}
		client.WithProject(app.GetProject())
		err = client.CreateProfile(name, put)
		return name, err
	}
	return "", nil

}

func (app *Compose) ProfileNameForService(service string) (string, error) {

	_, ok := app.Services[service]
	if !ok {
		return "", fmt.Errorf("service %s not found", service)
	}
	name := fmt.Sprintf("%s-%s-%s-cloudinit", app.Project, app.Name, service)

	return name, nil
}
func (app *Compose) ServiceNeedsInitProfile(service string) (bool, error) {
	svc, ok := app.Services[service]
	if !ok {
		return false, fmt.Errorf("service %s not found", service)
	}

	needsProfile := false

	if svc.CloudInitUserData != "" {
		needsProfile = true
	}
	if svc.CloudInitUserDataFile != "" {
		needsProfile = true
	}

	return needsProfile, nil

}

func (app *Compose) DeleteCloudProfileForService(service string) error {
	name, err := app.ProfileNameForService(service)
	if err != nil {
		return err
	}
	slog.Info("Deleting custom cloud-init profile", slog.String("instance", service), slog.String("profile", name))

	client, err := client.NewIncusClient()
	if err != nil {
		return err
	}
	client.WithProject(app.GetProject())
	return client.DeleteProfile(name)
}

func (app *Compose) InitContainerForService(service string) error {
	slog.Info("Initialize", slog.String("instance", service))
	/*
		svc, ok := app.Services[service]
		if !ok {
			return fmt.Errorf("service %s not found", service)
		}
		args := []string{"init", svc.Image, service}
		args = append(args, "--project", app.GetProject())

		for _, profile := range app.GetProfiles() {
			args = append(args, "--profile", profile)
		}
		for _, addProfile := range svc.AdditionalProfiles {
			args = append(args, "--profile", addProfile)
		}

		needsProfile, err := app.ServiceNeedsInitProfile(service)
		if err != nil {
			return err
		}
		if needsProfile {
			initProfile, err := app.CreateCloudProfileForService(service)
			if err != nil {
				return err
			}
			if initProfile != "" {
				args = append(args, "--profile", initProfile)
			}
		}
		if svc.EnvironmentFile != "" {
			args = append(args, "--environment-file", svc.EnvironmentFile)
		}
		if svc.Snapshot != nil {
			if svc.Snapshot.Schedule != "" {
				args = append(args, "--config=snapshots.schedule="+"\""+svc.Snapshot.Schedule+"\"")
			}
			if svc.Snapshot.Pattern != "" {
				args = append(args, "--config=snapshots.pattern="+"\""+svc.Snapshot.Pattern+"\"")
			}
			if svc.Snapshot.Expiry != "" {
				args = append(args, "--config=snapshots.expiry="+"\""+svc.Snapshot.Expiry+"\"")
			}
		}
		slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

		out, err := incus.ExecuteShellStream(context.Background(), args)
		if err != nil {
			slog.Error("Incus error", slog.String("message", out))
			return err
		}
		slog.Debug("Incus ", slog.String("message", out))
	*/

	var image string
	var remote string
	var iremote string
	var name string

	sc, err := app.ComposeProject.GetService(service)
	if err != nil {
		return err
	}
	client, err := client.NewIncusClient()
	if err != nil {
		return err
	}
	client.WithProject(app.GetProject())

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
			netName := fmt.Sprintf("eth%d", networkNumber)

			network, _, err := client.GetNetwork(net)
			if err != nil {
				return fmt.Errorf("failed loading network %q: %w", net, err)
			}

			// Prepare the instance's NIC device entry.
			var device map[string]string

			if network.Managed && client.HasExtension("instance_nic_network") {
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
		_, _, err := client.GetStoragePool(storageOverride)
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

	iremote, image, err = app.config.ParseRemote(sc.Image)
	if err != nil {
		return err
	}
	remote, name, err = app.config.ParseRemote(sc.Name)
	if err != nil {
		return err
	}
	iremote, image = guessImage(app.config, client.Client(), remote, iremote, image)
	// Deal with the default image
	if image == "" {
		image = "default"
	}
	imgRemote, imgInfo, err := getImgInfo(client.Client(), app.config, iremote, remote, image, &instancePost.Source)
	if err != nil {
		return err
	}

	if app.config.Remotes[iremote].Protocol == "incus" {

		instancePost.Type = api.InstanceType(imgInfo.Type)
	}
	fmt.Println(instancePost)

	op, err := client.Client().CreateInstanceFromImage(imgRemote, *imgInfo, instancePost)
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

// guessImage checks that the image name (provided by the user) is correct given an instance remote and image remote.
func guessImage(conf *config.Config, d incus.InstanceServer, instRemote string, imgRemote string, imageRef string) (string, string) {
	if instRemote != imgRemote {
		return imgRemote, imageRef
	}

	fields := strings.SplitN(imageRef, "/", 2)
	_, ok := conf.Remotes[fields[0]]
	if !ok {
		return imgRemote, imageRef
	}

	_, _, err := d.GetImageAlias(imageRef)
	if err == nil {
		return imgRemote, imageRef
	}

	_, _, err = d.GetImage(imageRef)
	if err == nil {
		return imgRemote, imageRef
	}

	if len(fields) == 1 {
		fmt.Fprintf(os.Stderr, "The local image '%q' couldn't be found, trying '%q:' instead."+"\n", imageRef, fields[0])
		return fields[0], "default"
	}

	fmt.Fprintf(os.Stderr, "The local image '%q' couldn't be found, trying '%q:%q' instead."+"\n", imageRef, fields[0], fields[1])
	return fields[0], fields[1]
}

// getImgInfo returns an image server and image info for the given image name (given by a user)
// an image remote and an instance remote.
func getImgInfo(d incus.InstanceServer, conf *config.Config, imgRemote string, instRemote string, imageRef string, source *api.InstanceSource) (incus.ImageServer, *api.Image, error) {
	var imgRemoteServer incus.ImageServer
	var imgInfo *api.Image
	var err error

	// Connect to the image server
	if imgRemote == instRemote {
		imgRemoteServer = d
	} else {
		imgRemoteServer, err = conf.GetImageServer(imgRemote)
		if err != nil {
			return nil, nil, err
		}
	}

	// Optimisation for public image servers.
	if conf.Remotes[imgRemote].Protocol != "incus" {
		imgInfo = &api.Image{}
		imgInfo.Fingerprint = imageRef
		imgInfo.Public = true
		source.Alias = imageRef
	} else {
		// Attempt to resolve an image alias
		alias, _, err := imgRemoteServer.GetImageAlias(imageRef)
		if err == nil {
			source.Alias = imageRef
			imageRef = alias.Target
		}

		// Get the image info
		imgInfo, _, err = imgRemoteServer.GetImage(imageRef)
		if err != nil {
			return nil, nil, err
		}
	}

	return imgRemoteServer, imgInfo, nil
}

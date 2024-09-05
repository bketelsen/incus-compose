package application

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"

	api "github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) CreateVolumesForService(service string) error {
	slog.Info("Creating Volumes", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for volName, vol := range svc.Volumes {
		fmt.Println("Creating volume", volName, vol)
		completeName := vol.CreateName(app.Name, service, volName)
		slog.Debug("Volume", slog.String("name", completeName), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		existingVolume, _ := app.showVolume(service, completeName, *vol)

		if existingVolume != nil && completeName == existingVolume.Name {
			slog.Info("Volume found", slog.String("volume", completeName))
		} else {
			fmt.Println("Creating volume", completeName, vol)
			err := app.createVolume(completeName, *vol)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (app *Compose) ListVolumesForService(service string) ([]string, error) {
	slog.Info("Getting Volumes", slog.String("instance", service))
	svc, ok := app.Services[service]
	if !ok {
		return []string{}, fmt.Errorf("service %s not found", service)
	}
	volumes := []string{}
	for volName, vol := range svc.Volumes {
		volumes = append(volumes, vol.CreateName(app.Name, service, volName)+" (pool: "+vol.Pool+")")
	}

	return volumes, nil
}

func (app *Compose) DeleteVolumesForService(service string) error {
	slog.Info("Deleting Volumes", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for volName, vol := range svc.Volumes {

		completeName := vol.CreateName(app.Name, service, volName)
		slog.Debug("Volume", slog.String("name", completeName), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		err := app.deleteVolume(completeName, *vol)
		if err != nil {
			return err
		}

	}

	return nil
}

func (app *Compose) AttachVolumesForService(service string) error {
	slog.Info("Attaching Volumes", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for volName, vol := range svc.Volumes {

		err := app.attachVolume(vol.CreateName(app.Name, service, volName), service, *vol)
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *Compose) createVolume(name string, vol Volume) error {
	slog.Info("Creating Volume", slog.String("volume", name))

	config := make(map[string]string)

	if vol.Snapshot != nil {
		if vol.Snapshot.Schedule != "" {
			config["snapshots.schedule"] = vol.Snapshot.Schedule
		}
		if vol.Snapshot.Pattern != "" {
			config["snapshots.pattern"] = vol.Snapshot.Pattern
		}
		if vol.Snapshot.Expiry != "" {
			config["snapshots.expiry"] = vol.Snapshot.Expiry
		}
	}

	// Parse the input
	volName, volType := parseVolume("custom", name)

	var volumePut api.StorageVolumePut

	// Create the storage volume entry
	newvol := api.StorageVolumesPost{
		Name:             volName,
		Type:             volType,
		ContentType:      "filesystem",
		StorageVolumePut: volumePut,
	}

	if volumePut.Config == nil {
		newvol.Config = map[string]string{}
	}

	for k, v := range config {
		newvol.Config[k] = v
	}
	// Parse remote
	resources, err := app.ParseServers(vol.Pool)
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf("Missing pool name")
	}

	client := resource.server
	err = client.CreateStoragePoolVolume(vol.Pool, newvol)
	if err != nil {
		return err
	}

	return nil
}

func (app *Compose) deleteVolume(name string, vol Volume) error {
	slog.Info("Deleting Volume", slog.String("volume", name))

	// Parse remote
	resources, err := app.ParseServers(vol.Pool)
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf("Missing pool name")
	}

	client := resource.server

	// Parse the input
	volName, volType := parseVolume("custom", vol.Name)
	fmt.Println("Deleting volume", volName, name, volType)

	// Delete the volume
	err = client.DeleteStoragePoolVolume(resource.name, volType, name)
	if err != nil {
		return err
	}

	fmt.Printf("Storage volume %s deleted"+"\n", vol.Name)

	return nil
}

func (app *Compose) attachVolume(name string, service string, vol Volume) error {
	slog.Info("Attaching Volume", slog.String("volume", name))

	args := []string{"storage", "volume", "attach", vol.Pool, name, service, vol.Mountpoint}
	args = append(args, "--project", app.GetProject())

	slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	d, err := app.getInstanceServer(service)
	if err != nil {
		return err
	}

	instance, etag, err := d.GetInstance(service)
	if err != nil {
		slog.Error(err.Error())
	}

	// Check if device exists
	_, ok := instance.Devices[name]
	if ok {
		slog.Info("Device already exists", slog.String("volume", name))
		return nil
	}

	volName, volType := parseVolume("custom", name)
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

	instance.Devices[name] = dev

	op, err := d.UpdateInstance(service, instance.Writable(), etag)
	if err != nil {
		return err
	}

	return op.Wait()
}

func (v *Volume) CreateName(application string, service string, volume string) string {
	return fmt.Sprintf("%s-%s-%s", application, service, volume)
}

func parseVolume(defaultType string, name string) (string, string) {
	parsedName := strings.SplitN(name, "/", 2)
	if len(parsedName) == 1 {
		return parsedName[0], defaultType
	} else if len(parsedName) == 2 && !slices.Contains([]string{"custom", "image", "container", "virtual-machine"}, parsedName[0]) {
		return name, defaultType
	}

	return parsedName[1], parsedName[0]
}

func (app *Compose) showVolume(service, name string, vol Volume) (*api.StorageVolume, error) {

	d, err := app.getInstanceServer(service)
	if err != nil {
		return nil, err
	}
	d.UseProject(app.GetProject())

	volName, volType := parseVolume("custom", name)

	volume, _, err := d.GetStoragePoolVolume(vol.Pool, volType, volName)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			if volType == "custom" {
				return nil, fmt.Errorf("Storage pool volume \"%s/%s\" not found. Try virtual-machine or container for type", volType, volName)
			}
			return nil, fmt.Errorf("Storage pool volume \"%s/%s\" notfound", volType, volName)
		}
		return nil, err
	}

	sort.Strings(volume.UsedBy)

	return volume, nil
}

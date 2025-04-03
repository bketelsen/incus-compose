package application

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/gosimple/slug"
	api "github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) CreateVolumesForService(service string) error {
	slog.Info("Creating Volumes", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	containerName := svc.GetContainerName()
	for volName, vol := range svc.Volumes {
		slog.Info("Creating volume", "name", volName)
		completeName := vol.CreateName(app.Name, containerName, volName)
		slog.Debug("Volume", slog.String("name", completeName), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		existingVolume, _ := app.showVolume(containerName, completeName, *vol)

		if existingVolume != nil && completeName == existingVolume.Name {
			slog.Info("Volume found", slog.String("volume", completeName))
		} else {
			slog.Info("Creating volume", "name", completeName)
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

	containerName := svc.GetContainerName()
	volumes := []string{}
	for volName, vol := range svc.Volumes {
		volumes = append(volumes, vol.CreateName(app.Name, containerName, volName)+" (pool: "+vol.Pool+")")
	}

	return volumes, nil
}

func (app *Compose) DeleteVolumesForService(service string) error {
	slog.Info("Deleting Volumes", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	containerName := svc.GetContainerName()

	for volName, vol := range svc.Volumes {

		completeName := vol.CreateName(app.Name, containerName, volName)
		slog.Debug("Volume", slog.String("name", completeName), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		existingVolume, _ := app.showVolume(containerName, completeName, *vol)

		if existingVolume == nil || completeName != existingVolume.Name {
			slog.Info("Volume not found", slog.String("volume", completeName))
		} else {
			err := app.deleteVolume(completeName, *vol)
			if err != nil {
				return err
			}
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
	containerName := svc.GetContainerName()
	for volName, vol := range svc.Volumes {

		err := app.attachVolume(vol.CreateName(app.Name, containerName, volName), service, *vol)
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

	if vol.Shift {
		config["security.shifted"] = "true"
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
		return fmt.Errorf("missing pool name")
	}

	client := resource.server.UseProject(app.GetProject())
	err = client.CreateStoragePoolVolume(vol.Pool, newvol)
	if err != nil {
		return err
	}

	return nil
}

func (app *Compose) deleteVolume(name string, vol Volume) error {

	// Parse remote
	resources, err := app.ParseServers(vol.Pool)
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf("missing pool name")
	}

	client := resource.server.UseProject(app.GetProject())

	// Parse the input
	volName, volType := parseVolume("custom", vol.Name)
	slog.Info("Deleting volume", "name", volName, "type", volType)

	// Delete the volume
	err = client.DeleteStoragePoolVolume(resource.name, volType, name)
	if err != nil {
		return err
	}

	return nil
}

func (app *Compose) attachVolume(name string, service string, vol Volume) error {
	slog.Info("Attaching Volume", slog.String("volume", name))

	args := []string{"storage", "volume", "attach", vol.Pool, name, service, vol.Mountpoint}
	args = append(args, "--project", app.GetProject())

	slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	containerName := svc.GetContainerName()

	d, err := app.getInstanceServer(containerName)
	if err != nil {
		return err
	}

	d = d.UseProject(app.GetProject())

	instance, etag, err := d.GetInstance(containerName)
	if err != nil {
		slog.Error(err.Error())
	}

	// Check if device exists
	_, ok = instance.Devices[name]
	if ok {
		slog.Info("Device already exists", slog.String("volume", name))
		return nil
	}

	volName, volType := parseVolume("custom", name)
	if volType != "custom" {
		return fmt.Errorf("only \"custom\" volumes can be attached to instances")
	}

	// Prepare the instance's device entry
	dev := map[string]string{
		"type":   "disk",
		"pool":   vol.Pool,
		"source": volName,
		"path":   vol.Mountpoint,
	}

	if vol.ReadOnly {
		dev["readonly"] = "true"
	}

	instance.Devices[name] = dev

	op, err := d.UpdateInstance(containerName, instance.Writable(), etag)
	if err != nil {
		return err
	}

	return op.Wait()
}

func (v *Volume) CreateName(application string, service string, volume string) string {
	name := slug.Make(fmt.Sprintf("%s-%s-%s", application, service, volume))
	if len(name) > 64 {
		sha256sum := sha256.Sum256([]byte(name))
		name = hex.EncodeToString(sha256sum[:16])
	}
	return name
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
	d = d.UseProject(app.GetProject())

	volName, volType := parseVolume("custom", name)

	volume, _, err := d.GetStoragePoolVolume(vol.Pool, volType, volName)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			if volType == "custom" {
				return nil, fmt.Errorf("storage pool volume \"%s/%s\" not found. Try virtual-machine or container for type", volType, volName)
			}
			return nil, fmt.Errorf("storage pool volume \"%s/%s\" notfound", volType, volName)
		}
		return nil, err
	}

	sort.Strings(volume.UsedBy)

	return volume, nil
}

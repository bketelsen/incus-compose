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
	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	d, err := app.getInstanceServer(service, app.GetProject())
	if err != nil {
		return err
	}
	inst, _, err := d.GetInstance(service)
	if err != nil {
		return err
	}

	containerName := inst.Name
	for _, vol := range svc.Volumes {
		slog.Info("Creating volume", "name", vol.Name)
		slog.Debug("Volume", slog.String("name", vol.Name), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		existingVolume, _ := app.showVolume(containerName, vol.Name, *vol)

		if existingVolume != nil && vol.Name == existingVolume.Name {
			slog.Info("Volume found", slog.String("volume", vol.Name))
		} else {
			slog.Info("Creating volume", "name", vol.Name)

			uid, ok := inst.Config["oci.uid"]
			if !ok {
				uid = "0"
			}

			gid, ok := inst.Config["oci.gid"]
			if !ok {
				gid = "0"
			}

			err := app.createVolume(vol.Name, *vol, uid, gid)
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
	for _, vol := range svc.Volumes {
		volumes = append(volumes, vol.Name+" (pool: "+vol.Pool+")")
	}

	return volumes, nil
}

func (app *Compose) DeleteVolumesForService(service string) error {
	slog.Info("Deleting Volumes", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	for _, vol := range svc.Volumes {

		slog.Debug("Volume", slog.String("name", vol.Name), slog.String("pool", vol.Pool), slog.String("mountpoint", vol.Mountpoint))

		existingVolume, _ := app.showVolume(service, vol.Name, *vol)

		if existingVolume == nil || vol.Name != existingVolume.Name {
			slog.Info("Volume not found", slog.String("volume", vol.Name))
		} else {
			err := app.deleteVolume(vol.Name, *vol)
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
	for _, vol := range svc.Volumes {

		err := app.attachVolume(vol.Name, service, *vol)
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *Compose) createVolume(name string, vol Volume, uid string, gid string) error {
	slog.Info("Creating Volume", "volume", name, "uid", uid, "gid", gid)

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

	config["security.shifted"] = "true"

	config["initial.uid"] = uid
	config["initial.gid"] = gid

	// Parse the input
	volName, volType := parseVolume("custom", name)

	// Create the storage volume entry
	newvol := api.StorageVolumesPost{
		Name:        volName,
		Type:        volType,
		ContentType: "filesystem",
		StorageVolumePut: api.StorageVolumePut{
			Config: config,
		},
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

	d, err := app.getInstanceServer(containerName, app.GetProject())
	if err != nil {
		return err
	}

	instance, etag, err := d.GetInstance(containerName)
	if err != nil {
		slog.Error(err.Error())
		return err
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

	d, err := app.getInstanceServer(service, app.GetProject())
	if err != nil {
		return nil, err
	}

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

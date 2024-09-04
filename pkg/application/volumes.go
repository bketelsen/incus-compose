package application

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
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

		// existingVolume, _ := app.showVolume(completeName, *vol)

		// if existingVolume != nil && completeName == existingVolume.Name {
		// 	slog.Info("Volume found", slog.String("volume", completeName))
		// } else {
		fmt.Println("Creating volume", completeName, vol)
		err := app.createVolume(completeName, *vol)
		if err != nil {
			return err
		}
		// }
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

	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
	}
	client.WithProject(app.GetProject())

	instance, _, err := client.GetInstance(service)
	if err != nil {
		slog.Error(err.Error())
	}

	// Check if device exists
	_, ok := instance.Devices[name]
	if ok {
		slog.Info("Device already exists", slog.String("volume", name))
		return nil
	}

	if err := client.AttachStorageVolume(vol.Pool, name, service, vol.Mountpoint); err != nil {
		slog.Error(err.Error())
	}

	return nil
}

func (app *Compose) showVolume(name string, vol Volume) (*api.StorageVolume, error) {
	slog.Info("Checking volume", slog.String("volume", name))

	args := []string{"storage", "volume", "show", vol.Pool, name}
	args = append(args, "--project", app.GetProject())

	slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
	}
	client.WithProject(app.GetProject())
	fmt.Println("showing volume", vol.Pool, name)
	v, err := client.ShowStorageVolume(vol.Pool, name)
	if err != nil {
		fmt.Println("showing volume error", err)
		return nil, err
	}

	return v, nil
}

func (app *Compose) ShowVolumesForService(service string) error {
	slog.Info("Showing", slog.String("instance", service))
	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	for volName, vol := range svc.Volumes {
		vol, err := app.showVolume(vol.CreateName(app.Name, service, volName), *vol)
		if err != nil {
			slog.Error(err.Error())
		}
		fmt.Println(vol)
	}
	return nil
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

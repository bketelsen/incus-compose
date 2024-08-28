package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bketelsen/incus-compose/pkg/incus"
	"github.com/bketelsen/incus-compose/pkg/incus/client"
	"github.com/lxc/incus/shared/api"
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
	if svc.Snapshot.Schedule != "" {
		args = append(args, "--config=snapshots.schedule="+"\""+svc.Snapshot.Schedule+"\"")
	}
	if svc.Snapshot.Pattern != "" {
		args = append(args, "--config=snapshots.pattern="+"\""+svc.Snapshot.Pattern+"\"")
	}
	if svc.Snapshot.Expiry != "" {
		args = append(args, "--config=snapshots.expiry="+"\""+svc.Snapshot.Expiry+"\"")
	}
	slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	out, err := incus.ExecuteShellStream(context.Background(), args)
	if err != nil {
		slog.Error("Incus error", slog.String("message", out))
		return err
	}
	slog.Debug("Incus ", slog.String("message", out))
	return nil

}

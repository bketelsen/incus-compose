package application

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
	"github.com/bketelsen/incus-compose/pkg/ui"
)

func (app *Compose) Up() error {

	for _, service := range app.Order(true) {

		err := app.InitContainerForService(service)
		if err != nil {
			return err
		}

		err = app.CreateVolumesForService(service)
		if err != nil {
			return err
		}

		err = app.CreateBindsForService(service)
		if err != nil {
			return err
		}

		// err = app.CreateGPUForService(service)
		// if err != nil {
		// 	return err
		// }
		err = app.AttachVolumesForService(service)
		if err != nil {
			return err
		}

		err = app.StartContainerForService(service, true)
		if err != nil {
			return err
		}

	}
	return nil
}

func (app *Compose) Stop(stateful, force bool, timeout int) error {
	for _, service := range app.Order(false) {

		err := app.StopContainerForService(service, stateful, force, timeout)
		if err != nil {
			return err
		}

	}
	return nil
}

func (app *Compose) Down(force, volumes bool, timeout int) error {
	for _, service := range app.Order(false) {

		err := app.StopContainerForService(service, false, force, timeout)
		if err != nil {
			return err
		}
		err = app.RemoveContainerForService(service, force)
		if err != nil {
			return err
		}
		if volumes {
			err = app.DeleteVolumesForService(service)
			if err != nil {
				return err
			}
		} else {
			vols, err := app.ListVolumesForService(service)
			if err != nil {
				return err
			}
			if len(vols) > 0 {
				for _, vol := range vols {
					slog.Warn("Volume not deleted", slog.String("instance", service), slog.String("volume", fmt.Sprintf("%v", vol)))
				}
			}
		}

		needsProfile, err := app.ServiceNeedsInitProfile(service)
		if err != nil {
			return err
		}
		if needsProfile {
			err = app.DeleteCloudProfileForService(service)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
func (app *Compose) Snapshot(noexpiry, stateful, volumes bool) error {
	for _, service := range app.Order(false) {
		slog.Info("Instance snapshot start", slog.String("instance", service))
		err := app.SnapshotInstance(service, noexpiry, stateful, volumes)
		if err != nil {
			return err
		}
		slog.Info("Instance snapshot complete", slog.String("instance", service))
		if volumes {
			for volName, vol := range app.Services[service].Volumes {
				slog.Info("Volume snapshot start", slog.String("volume", vol.CreateName(app.Name, service, volName)))
				err := app.SnapshotVolume(vol.Pool, vol.CreateName(app.Name, service, volName), noexpiry, stateful, volumes)
				if err != nil {
					return err
				}
				slog.Info("Volume snapshot complete", slog.String("volume", vol.CreateName(app.Name, service, volName)))
			}
		}

	}
	return nil
}

func (app *Compose) Export(volumes bool, customVolumesOnly bool) error {
	slog.Info("Export Root", slog.String("path", app.ExportPath))

	for _, service := range app.Order(false) {
		if !customVolumesOnly {
			slog.Info("Instance export start", slog.String("instance", service))
			err := app.ExportInstance(service, volumes)
			if err != nil {
				return err
			}
			slog.Info("Instance export complete", slog.String("instance", service))
		}
		if customVolumesOnly {
			for volName, vol := range app.Services[service].Volumes {
				slog.Info("Volume export start", slog.String("volume", vol.CreateName(app.Name, service, volName)))
				err := app.ExportVolume(vol.Pool, vol.CreateName(app.Name, service, volName))
				if err != nil {
					return err
				}
				slog.Info("Volume export complete", slog.String("volume", vol.CreateName(app.Name, service, volName)))
			}
		}

	}
	return nil
}

func (app *Compose) Start(wait bool) error {
	for _, service := range app.Order(true) {

		err := app.StartContainerForService(service, wait)
		if err != nil {
			return err
		}

	}
	return nil
}

func (app *Compose) Restart() error {
	for _, service := range app.Order(true) {

		err := app.RestartContainerForService(service)
		if err != nil {
			return err
		}

	}
	return nil
}

func (app *Compose) Remove(timeout int, force, stop, volumes bool) error {
	for _, service := range app.Order(false) {

		if stop {
			err := app.StopContainerForService(service, false, true, timeout)
			if err != nil {
				if strings.Contains(err.Error(), "already stopped") {
					slog.Info("Instance already stopped", slog.String("instance", service))
				} else {
					return err
				}
			}
		}
		err := app.RemoveContainerForService(service, force)
		if err != nil {
			if strings.Contains(err.Error(), "running") {
				slog.Error("Instance currently running", slog.String("instance", service))
				slog.Error("Stop it first or use --force", slog.String("instance", service))
				return err
			} else {
				return err
			}
		}
		if volumes {
			err = app.DeleteVolumesForService(service)
			if err != nil {
				return err
			}
		}
		// 	needsProfile, err := app.ServiceNeedsInitProfile(service)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	if needsProfile {
		// 		err = app.DeleteCloudProfileForService(service)
		// 		if err != nil {
		// 			return err
		// 		}
		// 	}
	}
	return nil
}

func (app *Compose) Info() error {

	instanceMap := make(map[string]ui.InstanceDetails)

	for service := range app.Services {

		client, err := client.NewIncusClient()
		if err != nil {
			slog.Error(err.Error())

			return err
		}
		client.WithProject(app.GetProject())
		i, _, err := client.GetInstance(service)
		if err != nil {
			slog.Error(err.Error())

			return err
		}
		s, _, err := client.GetInstanceState(service)
		if err != nil {
			slog.Error(err.Error())

			return err
		}
		instanceMap[service] = ui.InstanceDetails{Instance: i, State: s}

		// err = app.ShowContainerForService(service)
		// if err != nil {
		// 	return err
		// }

		// err = app.ShowDevicesForService(service)
		// if err != nil {
		// 	return err
		// }

	}
	ui.Info(instanceMap)

	return nil
}

func (app *Compose) Inventory() error {

	inventory := make(map[string][]string)
	defaultList := []string{}

	for service := range app.Services {
		svc, ok := app.Services[service]
		if !ok {
			return fmt.Errorf("service %s not found", service)
		}

		if len(svc.InventoryGroups) > 0 {
			for _, group := range svc.InventoryGroups {
				inventory[group] = append(inventory[group], service)
			}

		} else {
			defaultList = append(defaultList, service)

		}

	}

	Create := func(name, t string) *template.Template {
		return template.Must(template.New(name).Parse(t))
	}
	f, err := os.Create("hosts")
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl := Create("default", defaultTemplate)
	err = tmpl.Execute(f, defaultList)
	if err != nil {
		return err
	}
	tmpl2 := Create("group", group)
	err = tmpl2.Execute(f, inventory)
	if err != nil {
		return err
	}
	return nil
}

var defaultTemplate = `{{range .}}{{.}}
{{end -}}`
var group = `{{range $key, $value := .}}
[{{$key}}]
{{range $value}}{{.}}
{{end -}}
{{end -}}
`

package application

import (
	"fmt"
	"log/slog"
)

func (app *Compose) CreateBindsForService(service string) error {
	slog.Info("Creating BindMounts", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	containerName := svc.GetContainerName()
	for bindName, bind := range svc.BindMounts {

		slog.Debug("Bind", slog.Bool("shift", bind.Shift), slog.String("source", bind.Source), slog.String("target", bind.Target), slog.String("type", bind.Type))
		slog.Debug("Bind", slog.String("name", bindName))

		slog.Info("Creating BindMount", slog.String("name", bindName))

		device := map[string]string{}
		device["type"] = bind.Type
		device["source"] = bind.Source
		device["path"] = bind.Target
		if bind.Shift {
			device["shift"] = "true"
		}
		if bind.ReadOnly {
			device["readonly"] = "true"
		}

		// check for existing bind
		d, err := app.getInstanceServer(containerName)
		if err != nil {
			return err
		}
		d.UseProject(app.GetProject())

		inst, etag, err := d.GetInstance(containerName)
		if err != nil {
			return err
		}

		_, ok := inst.Devices[bindName]
		if ok {
			slog.Info("Device already exists", slog.String("name", bindName))
			return nil
		}

		inst.Devices[bindName] = device

		op, err := d.UpdateInstance(containerName, inst.Writable(), etag)
		if err != nil {
			return err
		}

		err = op.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

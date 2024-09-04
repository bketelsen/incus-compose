package application

import (
	"fmt"
	"log/slog"
)

func (app *Compose) CreateGPUForService(service string) error {

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	if svc.GPU {
		slog.Info("Adding GPU Device", slog.String("instance", service))

		err := app.createGPU(service)
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *Compose) createGPU(service string) error {
	slog.Info("Create GPU", slog.String("instance", service))
	bindName := service + "-gpu"

	slog.Info("Creating Device", slog.String("name", bindName))

	device := map[string]string{}
	device["type"] = "gpu"
	slog.Info("Creating BindMount", slog.String("name", bindName))

	err := app.addDevice(service, bindName, device)
	if err != nil {
		return err
	}

	return nil
}

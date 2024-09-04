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
		err := app.addDevice(service, bindName, device)
		if err != nil {
			return err
		}

	}

	return nil
}

// func (app *Compose) ShowDevicesForService(service string) error {
// 	slog.Info("Showing Device Info", slog.String("service", service))

// 	_, ok := app.Services[service]
// 	if !ok {
// 		return fmt.Errorf("service %s not found", service)
// 	}

// 	args := []string{"config", "device", "show", service}
// 	args = append(args, "--project", app.GetProject())

// 	out, err := incus.ExecuteShellStream(context.Background(), args)
// 	if err != nil {
// 		slog.Error("Incus error", slog.String("message", out))
// 		return err
// 	}
// 	slog.Debug("Incus ", slog.String("message", out))
// 	return nil
// }

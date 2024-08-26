package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/pkg/incus"
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

		args := []string{"config", "device", "add", service, bindName, bind.Type, "source=" + bind.Source, "path=" + bind.Target}
		if bind.Shift {
			args = append(args, "shift=true")
		}

		args = append(args, "--project", app.GetProject())

		slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

		out, err := incus.ExecuteShell(context.Background(), args)
		if err != nil {
			slog.Error("Incus error", slog.String("message", out))
			return err
		}
		slog.Debug("Incus ", slog.String("message", out))

	}

	return nil
}

func (app *Compose) ShowDevicesForService(service string) error {
	slog.Info("Showing Device Info", slog.String("service", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	args := []string{"config", "device", "show", service}
	args = append(args, "--project", app.GetProject())

	out, err := incus.ExecuteShellStream(context.Background(), args)
	if err != nil {
		slog.Error("Incus error", slog.String("message", out))
		return err
	}
	slog.Debug("Incus ", slog.String("message", out))
	return nil
}

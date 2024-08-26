package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/pkg/incus"
)

func (app *Compose) ShowContainerForService(service string) error {
	slog.Info("Showing", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	args := []string{"info", service}
	args = append(args, "--project", app.GetProject())

	slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	out, err := incus.ExecuteShellStream(context.Background(), args)
	if err != nil {
		slog.Error("Incus error", slog.String("message", out))
		return err
	}

	args = []string{"config", "show", service, "--expanded"}
	args = append(args, "--project", app.GetProject())

	slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	out, err = incus.ExecuteShellStream(context.Background(), args)
	if err != nil {
		slog.Error("Incus error", slog.String("message", out))
		return err
	}
	return nil

}

package application

import (
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
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

	// args := []string{"config", "device", "add", service, service + "-gpu", "gpu"}
	// args = append(args, "--project", app.GetProject())

	// slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	// out, err := incus.ExecuteShell(context.Background(), args)
	// if err != nil {
	// 	slog.Error("Incus error", slog.String("message", out))
	// 	return err
	// }
	// slog.Debug("Incus ", slog.String("message", out))
	bindName := service + "-gpu"

	slog.Info("Creating Device", slog.String("name", bindName))

	device := map[string]string{}
	device["type"] = "gpu"

	client, err := client.NewIncusClient()
	if err != nil {
		return err
	}
	client.WithProject(app.GetProject())
	err = client.AddDevice(service, bindName, device)
	if err != nil {
		return err
	}

	return nil
}

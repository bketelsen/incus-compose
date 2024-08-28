package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/pkg/incus"
	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) StartContainerForService(service string, wait bool) error {
	slog.Info("Starting", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	// args := []string{"start", service}
	// args = append(args, "--project", app.GetProject())

	// slog.Debug("Incus Args", slog.String("args", fmt.Sprintf("%v", args)))

	// out, err := incus.ExecuteShellStream(context.Background(), args)
	// if err != nil {
	// 	slog.Error("Incus error", slog.String("message", out))
	// 	return err
	// }
	// slog.Debug("Incus ", slog.String("message", out))

	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	client.WithProject(app.GetProject())
	err = client.InstanceAction("start", service, false, false)
	if err != nil {
		return err
	}

	if wait {
		if svc.CloudInitUserData != "" || svc.CloudInitUserDataFile != "" {
			slog.Info("cloud-init", slog.String("instance", service), slog.String("status", "waiting"))

			args := []string{"exec", service}
			args = append(args, "--project", app.GetProject())
			args = append(args, "--", "cloud-init", "status", "--wait")
			out, code, err := incus.ExecuteShellStreamExitCode(context.Background(), args)
			if err != nil {
				slog.Error("Incus error", slog.String("instance", service), slog.String("message", out))
				return err
			}
			if code == 2 {
				slog.Error("cloud-init", slog.String("instance", service), slog.String("status", "completed with recoverable errors"))
			}

			slog.Info("cloud-init", slog.String("instance", service), slog.String("status", "done"))
			slog.Debug("Incus ", slog.String("instance", service), slog.String("message", out))
		}
	}
	return nil
}

func (app *Compose) RestartContainerForService(service string) error {
	slog.Info("Restarting", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	client.WithProject(app.GetProject())
	return client.InstanceAction("restart", service, false, false)

}

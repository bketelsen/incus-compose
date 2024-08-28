package application

import (
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) StopContainerForService(service string, stateful, force bool) error {
	slog.Info("Stopping", slog.String("instance", service))

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
	return client.InstanceAction("stop", service, stateful, force)

}

package application

import (
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) RemoveContainerForService(service string) error {
	slog.Info("Removing", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	client, err := client.NewIncusClient()
	if err != nil {
		return err
	}
	client.WithProject(app.GetProject())

	inst, _, _ := client.GetInstance(service)
	if inst != nil && inst.Name == service {
		return client.DeleteInstance(service)
	}
	return nil

}

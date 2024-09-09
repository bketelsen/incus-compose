package application

import (
	"fmt"
	"log/slog"
	"time"

	api "github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) SnapshotInstance(service string, noexpiry, stateful, volumes bool) error {
	slog.Info("Showing", slog.String("instance", service))
	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	containerName := svc.GetContainerName()
	return app.createSnapshot(containerName, snapshotName(containerName), stateful, noexpiry, time.Now().Add(time.Hour*24*7))

}

func (app *Compose) createSnapshot(instanceName, snapshotName string, stateful bool, noexpiry bool, expiration time.Time) error {
	d, err := app.getInstanceServer(instanceName)
	if err != nil {
		return err
	}

	d.UseProject(app.GetProject())

	req := api.InstanceSnapshotsPost{
		Name:     snapshotName,
		Stateful: stateful,
	}

	if noexpiry {
		req.ExpiresAt = &time.Time{}
	} else if !expiration.IsZero() {
		req.ExpiresAt = &expiration
	}

	op, err := d.CreateInstanceSnapshot(instanceName, req)
	if err != nil {
		return err
	}

	return op.Wait()

}

func snapshotName(resource string) string {
	return resource + "-" + time.Now().Format("2006-01-02-15-04-05")
}

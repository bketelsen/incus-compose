package application

import (
	"log/slog"
	"time"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) SnapshotInstance(service string, noexpiry, stateful, volumes bool) error {
	slog.Info("Showing", slog.String("instance", service))

	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	client.WithProject(app.GetProject())
	return client.SnapshotInstance(service, snapshotName(service), stateful, noexpiry, time.Now().Add(time.Hour*24*7))

}

func snapshotName(resource string) string {
	return resource + "-" + time.Now().Format("2006-01-02-15-04-05")
}

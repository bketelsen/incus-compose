package application

import (
	"log/slog"
	"time"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) SnapshotVolume(pool, volume string, noexpiry, stateful, volumes bool) error {

	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	client.WithProject(app.GetProject())
	return client.SnapshotVolume(pool, volume, snapshotName(volume), stateful, noexpiry, time.Now().Add(time.Hour*24*7))

}

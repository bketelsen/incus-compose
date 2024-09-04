package application

import (
	"fmt"
	"time"

	api "github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) SnapshotVolume(pool, volume string, noexpiry, stateful, volumes bool) error {

	return app.volumeSnapshot(pool, volume, snapshotName(volume), stateful, noexpiry, time.Now().Add(time.Hour*24*7))

}

func (app *Compose) volumeSnapshot(pool, volume, snapshotName string, stateful bool, noexpiry bool, expiration time.Time) error {
	// Parse remote
	resources, err := app.ParseServers(pool)
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf("Missing pool name")
	}
	req := api.StorageVolumeSnapshotsPost{
		Name: snapshotName,
	}

	if noexpiry {
		req.ExpiresAt = &time.Time{}
	} else if !expiration.IsZero() {
		req.ExpiresAt = &expiration
	}

	op, err := resource.server.CreateStoragePoolVolumeSnapshot(pool, "custom", volume, req)
	if err != nil {
		return err
	}

	return op.Wait()

}

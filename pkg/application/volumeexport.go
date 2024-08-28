package application

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) ExportVolume(pool, volume string) error {

	slog.Info("Exporting", slog.String("volume", volume))

	client, err := client.NewIncusClient()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	client.WithProject(app.GetProject())
	// make sure app.ExportPath exists
	if err := os.MkdirAll(app.ExportPath, 0755); err != nil {
		return err
	}
	fullExportPath := filepath.Join(app.ExportPath, exportName(volume))
	slog.Info("Export File", slog.String("path", fullExportPath))

	return client.ExportVolume(pool, volume, fullExportPath)

}

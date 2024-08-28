package application

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/bketelsen/incus-compose/pkg/incus/client"
)

func (app *Compose) ExportInstance(service string, volumes bool) error {
	slog.Info("Exporting", slog.String("instance", service))

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
	fullExportPath := filepath.Join(app.ExportPath, exportName(service))
	slog.Info("Export File", slog.String("path", fullExportPath))

	return client.ExportInstance(service, fullExportPath, !volumes)

}

func exportName(resource string) string {
	return resource + "-" + "export" + "-" + time.Now().Format("2006-01-02-15-04-05") + ".tar.gz"
}

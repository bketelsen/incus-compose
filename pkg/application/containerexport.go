package application

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	incus "github.com/lxc/incus/v6/client"
	api "github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) ExportInstance(service string, volumes bool) error {
	slog.Info("Exporting", slog.String("instance", service))

	fullExportPath := filepath.Join(app.ExportPath, exportName(service))
	slog.Info("Export File", slog.String("path", fullExportPath))

	return app.instanceExport(service, fullExportPath, !volumes)

}

func exportName(resource string) string {
	return resource + "-" + "export" + "-" + time.Now().Format("2006-01-02-15-04-05") + ".tar.gz"
}

func (app *Compose) instanceExport(instanceName, targetName string, instanceOnly bool) error {
	d, err := app.getInstanceServer(instanceName)
	if err != nil {
		return err
	}
	req := api.InstanceBackupsPost{
		Name:             "",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		InstanceOnly:     instanceOnly,
		OptimizedStorage: false,
	}

	op, err := d.CreateInstanceBackup(instanceName, req)
	if err != nil {
		return fmt.Errorf("create instance backup: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return err
	}
	// Get name of backup
	uStr := op.Get().Resources["backups"][0]
	u, err := url.Parse(uStr)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", uStr, err)
	}

	backupName, err := url.PathUnescape(path.Base(u.EscapedPath()))
	if err != nil {
		return fmt.Errorf("invalid backup name segment in path %q: %w", u.EscapedPath(), err)
	}

	defer func() {
		// Delete backup after we're done
		op, err = d.DeleteInstanceBackup(instanceName, backupName)
		if err == nil {
			_ = op.Wait()
		}
	}()

	var target *os.File

	target, err = os.Create(targetName)
	if err != nil {
		return err
	}

	defer func() { _ = target.Close() }()

	backupFileRequest := incus.BackupFileRequest{
		BackupFile: io.WriteSeeker(target),
	}
	_, err = d.GetInstanceBackupFile(instanceName, backupName, &backupFileRequest)
	if err != nil {
		_ = os.Remove(targetName)
		return fmt.Errorf("fetch instance backup file: %w", err)
	}
	err = target.Close()
	if err != nil {
		return fmt.Errorf("failed to close export file: %w", err)
	}

	return nil
}

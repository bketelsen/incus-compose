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

func (app *Compose) ExportVolume(pool, volume string) error {

	slog.Info("Exporting", slog.String("volume", volume))

	fullExportPath := filepath.Join(app.ExportPath, exportName(volume))
	slog.Info("Export File", slog.String("path", fullExportPath))

	return app.volumeExport(pool, volume, fullExportPath)

}

func (app *Compose) volumeExport(pool, volume, targetName string) error {
	// Parse remote
	resources, err := app.ParseServers(pool)
	if err != nil {
		return err
	}

	resource := resources[0]
	if resource.name == "" {
		return fmt.Errorf("missing pool name")
	}
	req := api.StorageVolumeBackupsPost{
		Name:             "",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		VolumeOnly:       true,
		OptimizedStorage: false,
	}
	resource.server.UseProject(app.GetProject())
	op, err := resource.server.CreateStorageVolumeBackup(pool, volume, req)
	if err != nil {
		return fmt.Errorf("failed to create storage volume backup: %w", err)
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
		op, err = resource.server.DeleteStorageVolumeBackup(pool, volume, backupName)
		if err == nil {
			_ = op.Wait()
		}
	}()

	target, err := os.Create(targetName)
	if err != nil {
		return err
	}

	defer func() { _ = target.Close() }()

	backupFileRequest := incus.BackupFileRequest{
		BackupFile: io.WriteSeeker(target),
	}

	// Export tarball
	_, err = resource.server.GetStorageVolumeBackupFile(pool, volume, backupName, &backupFileRequest)
	if err != nil {
		_ = os.Remove(targetName)
		return fmt.Errorf("failed to fetch storage volume backup file: %w", err)
	}

	return nil
}

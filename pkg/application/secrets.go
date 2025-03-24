package application

import (
	"fmt"
	"log/slog"
	"os"
)

func (app *Compose) CreateSecretsForService(service string) error {
	slog.Info("Creating Secrets", slog.String("instance", service))

	svc, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}
	containerName := svc.GetContainerName()

	// add secrets files
	if len(svc.Secrets) == 0 {
		return nil
	}

	for k := range svc.Secrets {
		slog.Debug("Adding Secret", slog.String("instance", service), slog.String("secret name", k))
		secretsFileId := fmt.Sprintf("%s_%s", app.Name, k)

		sf, ok := app.SecretsFiles[secretsFileId]
		if ok {
			// create local secret file in the /.secrets/serviceName/ directory
			secPath := fmt.Sprintf(".secrets/%s/%s", service, k)
			dirPath := fmt.Sprintf(".secrets/%s", service)

			f, err := os.ReadFile(sf.FilePath)
			if err != nil {
				return err
			}

			if _, err := os.Stat(secPath); os.IsNotExist(err) {
				if err = os.MkdirAll(dirPath, 0755); err != nil {
					return err
				}

				err = os.WriteFile(secPath, f, 0644)
				if err != nil {
					return err
				}

			} else {
				// rewrite th secret file content
				file, err := os.Create(secPath)
				if err != nil {
					return err
				}
				defer file.Close()

				if _, err = file.Write(f); err != nil {
					return err
				}

				if err = file.Sync(); err != nil {
					return err
				}

			}

			absPath, err := os.Getwd()
			if err != nil {
				return err
			}
			absPath = fmt.Sprintf("%s/%s", absPath, dirPath)
			mntPath := "/run/secrets"
			bindName := fmt.Sprintf("secrets-%s", service)

			// check for existing bind
			d, err := app.getInstanceServer(containerName)
			if err != nil {
				return err
			}
			d.UseProject(app.GetProject())

			inst, _, err := d.GetInstance(containerName)
			if err != nil {
				return err
			}

			_, ok := inst.Devices[bindName]
			if ok {
				slog.Info("Device already exists", slog.String("name", bindName))
				return nil
			}

			device := map[string]string{}
			device["type"] = "disk"
			device["source"] = absPath
			device["path"] = mntPath

			err = app.addDevice(service, bindName, device)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

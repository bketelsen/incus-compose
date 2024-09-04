package application

import (
	"fmt"
	"log/slog"

	"github.com/lxc/incus/v6/shared/api"
)

func (app *Compose) RemoveContainerForService(service string, force bool) error {
	slog.Info("Removing", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	return app.removeInstance(service, force)

}
func (app *Compose) StopContainerForService(service string, stateful, force bool, timeout int) error {
	slog.Info("Stopping", slog.String("instance", service))

	_, ok := app.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found", service)
	}

	return app.updateInstanceState(service, "stop", timeout, force, stateful)

}

func (app *Compose) updateInstanceState(name string, state string, timeout int, force bool, stateful bool) error {
	remote, name, err := app.conf.ParseRemote(name)
	if err != nil {
		return err
	}

	d, err := app.conf.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	req := api.InstanceStatePut{
		Action:   state,
		Timeout:  timeout,
		Force:    force,
		Stateful: stateful,
	}

	op, err := d.UpdateInstanceState(name, req, "")
	if err != nil {
		return err
	}

	return op.Wait()
}

func (app *Compose) removeInstance(name string, force bool) error {

	// Parse remote
	resources, err := app.ParseServers(name)
	if err != nil {
		return err
	}

	// Check that everything exists.
	err = instancesExist(resources)
	if err != nil {
		return err
	}

	// Process with deletion.
	for _, resource := range resources {
		connInfo, err := resource.server.GetConnectionInfo()
		if err != nil {
			return err
		}

		ct, _, err := resource.server.GetInstance(resource.name)
		if err != nil {
			return err
		}

		if ct.StatusCode != 0 && ct.StatusCode != api.Stopped {
			if !force {
				return fmt.Errorf("The instance is currently running, stop it first or pass --force")
			}

			req := api.InstanceStatePut{
				Action:  "stop",
				Timeout: -1,
				Force:   true,
			}

			op, err := resource.server.UpdateInstanceState(resource.name, req, "")
			if err != nil {
				return err
			}

			err = op.Wait()
			if err != nil {
				return fmt.Errorf("Stopping the instance failed: %s", err)
			}

			if ct.Ephemeral {
				continue
			}
		}

		// if c.flagForceProtected && util.IsTrue(ct.ExpandedConfig["security.protection.delete"]) {
		// 	// Refresh in case we had to stop it above.
		// 	ct, etag, err := resource.server.GetInstance(resource.name)
		// 	if err != nil {
		// 		return err
		// 	}

		// 	ct.Config["security.protection.delete"] = "false"
		// 	op, err := resource.server.UpdateInstance(resource.name, ct.Writable(), etag)
		// 	if err != nil {
		// 		return err
		// 	}

		// 	err = op.Wait()
		// 	if err != nil {
		// 		return err
		// 	}
		// }

		// Instance delete
		op, err := resource.server.DeleteInstance(name)
		if err != nil {
			return fmt.Errorf("Failed deleting instance %q in project %q: %w", resource.name, connInfo.Project, err)
		}

		return op.Wait()
	}
	return nil

}

package application

import (
	"log/slog"
	"strings"
)

func (c *Compose) ListServices() []string {

	return c.ComposeProject.ServiceNames()
}
func (c *Compose) DependentsForService(s string) ([]string, error) {

	sc, err := c.ComposeProject.GetService(s)
	if err != nil {
		return nil, err
	}
	return c.ComposeProject.GetDependentsForService(sc), nil

}
func (c *Compose) StopService(s string, stateful, force bool, timeout int) error {
	return c.StopContainerForService(s, stateful, force, timeout)
}
func (c *Compose) StartService(s string, wait bool) error {
	return c.StartContainerForService(s, wait)
}
func (c *Compose) StopAll(stateful, force bool, timeout int) error {
	ss := c.ListServices()
	for _, s := range ss {
		err := c.StopContainerForService(s, stateful, force, timeout)
		if err != nil {
			if strings.Contains(err.Error(), "already stopped") {
				slog.Info("Instance already stopped", slog.String("instance", s))
			} else {
				return err
			}
		}
	}
	return nil
}
func (c *Compose) StartAll(wait bool) error {
	ss := c.ListServices()
	for _, s := range ss {
		err := c.StartContainerForService(s, wait)
		if err != nil {
			if strings.Contains(err.Error(), "already running") {
				slog.Info("Instance already running", slog.String("instance", s))
			} else {
				return err
			}
		}
	}
	return nil
}

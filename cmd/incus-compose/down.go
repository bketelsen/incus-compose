/*
Copyright © 2024 Brian Ketelsen <bketelsen@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"fmt"
	"log/slog"

	"github.com/bketelsen/incus-compose/internal/i18n"
	"github.com/spf13/cobra"
)

type cmdDown struct {
	global       *cmdGlobal
	action       *cmdAction
	delete       *cmdDelete
	volumeDelete *cmdStorageVolumeDelete

	flagForce   bool
	flagStop    bool
	flagVolumes bool
	flagTimeout int
}

// downCmd represents the down command
func (c *cmdDown) Command() *cobra.Command {
	cmdAction := cmdAction{global: c.global}
	c.action = &cmdAction
	cmdDelete := cmdDelete{global: c.global}
	c.delete = &cmdDelete
	c.volumeDelete = &cmdStorageVolumeDelete{global: c.global}

	cmd := &cobra.Command{}
	cmd.Use = "down"
	cmd.Short = "Stop and remove instances"
	cmd.Long = `Stop and remove instances

Optionally remove custom storage volumes declared in the compose file, with the --volumes flag.
`
	cmd.RunE = c.Run
	cmd.Flags().BoolVarP(&c.flagStop, "stop", "s", false, i18n.G("Stop running instances before removing"))
	cmd.Flags().BoolVarP(&c.flagForce, "force", "f", false, i18n.G("Don't ask for confirmation before removing instances"))
	cmd.Flags().BoolVar(&c.flagVolumes, "volumes", false, i18n.G("Remove named volumes declared in the 'volumes' section of the compose file"))
	cmd.Flags().IntVarP(&c.flagTimeout, "timeout", "t", -1, i18n.G("Specify a shutdown timeout in seconds"))

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return c.global.cmpInstances(toComplete)
	}

	return cmd
}

// downCmd represents the down command
func (c *cmdDown) Run(cmd *cobra.Command, args []string) error {

	for _, service := range c.global.compose.Order(false) {

		if c.flagStop {
			err := c.action.doAction("stop", c.global.conf, service)
			if err != nil {
				return err
			}
			if !c.global.flagQuiet {
				fmt.Printf(i18n.G("Instance %s stopped")+"\n", service)
			}

		}
		c.delete.flagForce = c.flagForce
		err := c.delete.Run(cmd, []string{service})
		if err != nil {
			return err
		}
		if !c.global.flagQuiet {
			fmt.Printf(i18n.G("Instance %s deleted")+"\n", service)
		}

		for name, vol := range c.global.compose.Services[service].Volumes {
			if c.flagVolumes {
				err := c.volumeDelete.Run(vol.Pool, vol.Name(c.global.compose.Name, service, name), "")
				if err != nil {
					return err
				}

			} else {
				slog.Warn("Volume not deleted", slog.String("instance", service), slog.String("volume", fmt.Sprintf("%v", vol)))

			}

		}

	}
	return nil
}

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
package cmd

import (
	"fmt"
	"log/slog"

	"github.com/bketelsen/toolbox/cobra"
)

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove instances",
	Long: `Stop and remove instances

Optionally remove custom storage volumes declared in the compose file, with the --volumes flag.
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Logger.Info("Down", slog.String("app", app.Name))

		err := app.Down(cmd.Flag("force").Changed, cmd.Flag("volumes").Changed, timeout)
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().BoolP("force", "f", false, "Don't ask for confirmation before removing instances")
	downCmd.Flags().BoolP("volumes", "v", false, "Remove named volumes declared in the 'volumes' section of the compose file")
	downCmd.Flags().IntVarP(&timeout, "timeout", "t", -1, "Specify a shutdown timeout in seconds")

}

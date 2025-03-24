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

// snapshotCmd represents the backup command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create snapshots of instances and volumes",
	Long:  `Create snapshots of instances and volumes`,
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Snapshotting", slog.String("app", app.Name))

		err := app.Snapshot(cmd.Flag("noexpiry").Changed, cmd.Flag("stateful").Changed, cmd.Flag("volumes").Changed)
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)

	snapshotCmd.Flags().BoolP("noexpiry", "n", false, "No expiry date for the snapshot")
	snapshotCmd.Flags().BoolP("stateful", "s", false, "Stateful snapshot, if supported")
	snapshotCmd.Flags().BoolP("volumes", "v", false, "Snapshot volumes")

}

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
	"log/slog"

	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:  "rm",
	Args: cobra.MaximumNArgs(1),

	Short: "Remove stopped instances",
	Long: `Remove stopped instances
	
Remove stopped instances declared in the compose file. By default, volumes
declared in the compose file are not removed. You can override this with the
--volumes flag.`,
	Run: func(cmd *cobra.Command, args []string) {

		slog.Info("Removing", slog.String("app", app.Name))

		err := app.Remove(timeout, cmd.Flag("force").Changed, cmd.Flag("stop").Changed, cmd.Flag("volumes").Changed)
		if err != nil {
			slog.Error("Remove", slog.String("error", err.Error()))
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("force", "f", false, "Don't ask for confirmation before removing instances")
	rmCmd.Flags().BoolP("stop", "s", false, "Stop the instances, if required, before removing")
	rmCmd.Flags().BoolP("volumes", "v", false, "Remove named volumes declared in the compose file")
	rmCmd.Flags().IntP("timeout", "t", 0, "Specify a shutdown timeout in seconds")
}

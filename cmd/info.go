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

	"github.com/bketelsen/toolbox/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:  "info",
	Args: cobra.MaximumNArgs(1),

	Short: "Display information about instances",
	Long:  `Display information about instances`,
	Run: func(cmd *cobra.Command, args []string) {

		slog.Info("Info", slog.String("app", app.Name))

		err := app.Info()
		if err != nil {
			slog.Error("Info", slog.String("error", err.Error()))
		}

	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

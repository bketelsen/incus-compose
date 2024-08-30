package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/lxc/incus/v6/shared/api"
)

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
	white     = lipgloss.Color("255")
)

type InstanceDetails struct {
	Instance *api.Instance
	State    *api.InstanceState
}

func Info(instanceMap map[string]InstanceDetails) {
	re := lipgloss.NewRenderer(os.Stdout)
	var (
		// HeaderStyle is the lipgloss style used for the table headers.
		HeaderStyle = re.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
		// CellStyle is the base lipgloss style used for the table rows.
		CellStyle = re.NewStyle().Padding(0, 1).Width(14)
		// OddRowStyle is the lipgloss style used for odd-numbered table rows.
		OddRowStyle = CellStyle.Foreground(lightGray)
		// EvenRowStyle is the lipgloss style used for even-numbered table rows.
		EvenRowStyle = CellStyle.Foreground(white)
		// BorderStyle is the lipgloss style used for the table border.
		BorderStyle = lipgloss.NewStyle().Foreground(purple)
	)

	t := table.New().
		Border(lipgloss.ThickBorder()).
		BorderStyle(BorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = EvenRowStyle
			default:
				style = OddRowStyle
			}

			// Make the second column a little wider.
			if col == 1 {
				style = style.Width(52)
			}

			return style
		}).
		Headers("Instance", "Details", "Status") // This function is a placeholder for the package documentation.
	for service, details := range instanceMap {
		deets := strings.Builder{}
		// add Instance information
		deets.WriteString(fmt.Sprintf("Type: %s\n", details.Instance.Type))
		// add Project information
		deets.WriteString(fmt.Sprintf("Project: %s\n", details.Instance.Project))

		// add Device Information
		deets.WriteString("Devices:\n")

		deviceInfo := ""
		if details.Instance.ExpandedDevices != nil {
			for devName, dev := range details.Instance.ExpandedDevices {
				deviceInfo += fmt.Sprintf("  %s:\n", devName)
				for key, val := range dev {
					deviceInfo += fmt.Sprintf("    %s: %s\n", key, val)
				}
			}

		}
		deets.WriteString(deviceInfo)

		deets.WriteString("Network:\n")

		// add Network information
		networkInfo := ""
		if details.State.Network != nil {
			for netName, net := range details.State.Network {
				if netName == "eth0" {
					networkInfo += fmt.Sprintf("  %s:\n", netName)

					networkInfo += fmt.Sprintf("    %s:\n", "IP addresses")

					for _, addr := range net.Addresses {
						if addr.Family == "inet" {
							networkInfo += fmt.Sprintf("      %s:  %s/%s (%s)\n", addr.Family, addr.Address, addr.Netmask, addr.Scope)
						} else {
							networkInfo += fmt.Sprintf("      %s: %s/%s (%s)\n", addr.Family, addr.Address, addr.Netmask, addr.Scope)
						}
					}
				}
			}
		}
		deets.WriteString(networkInfo)
		t.Row(service, deets.String(), details.State.Status)
	}

	fmt.Println(t)
}

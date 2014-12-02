package main

import (
	"fmt"

	"github.com/cloudfoundry/cli/plugin"
)

func main() {
	plugin.Start(&AutopilotPlugin{})
}

type AutopilotPlugin struct{}

func (AutopilotPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	fmt.Println("what up yo")
}

func (AutopilotPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "Autopilot",
		Commands: []plugin.Command{
			{
				Name:     "zero-downtime-push",
				HelpText: "Perform a zero-downtime push of an application over the top of an old one",
			},
		},
	}
}

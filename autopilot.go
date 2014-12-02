package main

import (
	"errors"
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

var ErrNoManifest = errors.New("a manifest is required to push this application")

type ApplicationRepo struct {
	conn plugin.CliConnection
}

func NewApplicationRepo(conn plugin.CliConnection) *ApplicationRepo {
	return &ApplicationRepo{
		conn: conn,
	}
}

func (repo *ApplicationRepo) RenameApplication(oldName, newName string) error {
	_, err := repo.conn.CliCommand("rename", oldName, newName)
	return err
}

func (repo *ApplicationRepo) PushApplication(manifestPath, appPath string) error {
	if manifestPath == "" {
		return ErrNoManifest
	}

	args := []string{"push", "-f", manifestPath}

	if appPath != "" {
		args = append(args, "-p", appPath)
	}

	_, err := repo.conn.CliCommand(args...)
	return err
}

func (repo *ApplicationRepo) DeleteApplication(appName string) error {
	_, err := repo.conn.CliCommand("delete", appName, "-f", "-r")
	return err
}

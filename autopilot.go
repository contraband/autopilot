package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/cli/plugin"
)

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		os.Exit(1)
	}
}

func main() {
	plugin.Start(&AutopilotPlugin{})
}

type AutopilotPlugin struct{}

func (plugin AutopilotPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	appRepo := NewApplicationRepo(cliConnection)

	appName := os.Args[3]

	venerableAppName := appName + "-venerable"

	err := appRepo.RenameApplication(appName, venerableAppName)
	fatalIf(err)

	err = appRepo.PushApplication(os.Args[3:])
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		err = appRepo.DeleteApplication(appName)
		fatalIf(err)
		err := appRepo.RenameApplication(venerableAppName, appName)
		fatalIf(err)
		os.Exit(1)
	}

	err = appRepo.DeleteApplication(venerableAppName)
	fatalIf(err)

	fmt.Println()
	fmt.Println("A new version of your application has successfully been pushed!")
	fmt.Println()

	err = appRepo.ListApplications()
	fatalIf(err)
}

func (AutopilotPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "autopilot",
		Commands: []plugin.Command{
			{
				Name:     "zero-downtime-push",
				HelpText: "Perform a zero-downtime push of an application over the top of an old one",
			},
		},
	}
}

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

func (repo *ApplicationRepo) PushApplication(args []string) error {
	pushArgs := args
	pushArgs = append([]string{"push"}, pushArgs...)

	_, err := repo.conn.CliCommand(pushArgs...)
	return err
}

func (repo *ApplicationRepo) DeleteApplication(appName string) error {
	_, err := repo.conn.CliCommand("delete", appName, "-f")
	return err
}

func (repo *ApplicationRepo) ListApplications() error {
	_, err := repo.conn.CliCommand("apps")
	return err
}

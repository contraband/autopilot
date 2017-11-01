package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/api/logs"
	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/contraband/autopilot/rewind"
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

func venerableAppName(appName string) string {
	return fmt.Sprintf("%s-venerable", appName)
}

func getActionsForExistingApp(appRepo *ApplicationRepo, appName, manifestPath, appPath string, showLogs bool) []rewind.Action {
	return []rewind.Action{
		// rename
		{
			Forward: func() error {
				return appRepo.RenameApplication(appName, venerableAppName(appName))
			},
		},
		// push
		{
			Forward: func() error {
				return appRepo.PushApplication(appName, manifestPath, appPath, showLogs)
			},
			ReversePrevious: func() error {
				// If the app cannot start we'll have a lingering application
				// We delete this application so that the rename can succeed
				appRepo.DeleteApplication(appName)

				return appRepo.RenameApplication(venerableAppName(appName), appName)
			},
		},
		// delete
		{
			Forward: func() error {
				return appRepo.DeleteApplication(venerableAppName(appName))
			},
		},
	}
}

func getActionsForNewApp(appRepo *ApplicationRepo, appName, manifestPath, appPath string, showLogs bool) []rewind.Action {
	return []rewind.Action{
		// push
		{
			Forward: func() error {
				return appRepo.PushApplication(appName, manifestPath, appPath, showLogs)
			},
		},
	}
}

func (plugin AutopilotPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) > 0 && args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	appRepo := NewApplicationRepo(cliConnection)
	appName, manifestPath, appPath, showLogs, err := ParseArgs(args)
	fatalIf(err)

	appExists, err := appRepo.DoesAppExist(appName)
	fatalIf(err)

	var actionList []rewind.Action

	if appExists {
		actionList = getActionsForExistingApp(appRepo, appName, manifestPath, appPath, showLogs)
	} else {
		actionList = getActionsForNewApp(appRepo, appName, manifestPath, appPath, showLogs)
	}

	actions := rewind.Actions{
		Actions:              actionList,
		RewindFailureMessage: "Oh no. Something's gone wrong. I've tried to roll back but you should check to see if everything is OK.",
	}

	err = actions.Execute()
	fatalIf(err)

	fmt.Println()
	fmt.Println("A new version of your application has successfully been pushed!")
	fmt.Println()

	_ = appRepo.ListApplications()
}

func (AutopilotPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "autopilot",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
			Build: 4,
		},
		Commands: []plugin.Command{
			{
				Name:     "zero-downtime-push",
				HelpText: "Perform a zero-downtime push of an application over the top of an old one",
				UsageDetails: plugin.Usage{
					Usage: "$ cf zero-downtime-push application-to-replace \\ \n \t-f path/to/new_manifest.yml \\ \n \t-p path/to/new/path",
				},
			},
		},
	}
}

func ParseArgs(args []string) (string, string, string, bool, error) {
	flags := flag.NewFlagSet("zero-downtime-push", flag.ContinueOnError)
	manifestPath := flags.String("f", "", "path to an application manifest")
	appPath := flags.String("p", "", "path to application files")
	showLogs := flags.Bool("show-app-log", false, "tail and show application log during application start")

	if len(args) < 2 {
		return "", "", "", false, ErrNoArgs
	}
	err := flags.Parse(args[2:])
	if err != nil {
		return "", "", "", false, err
	}

	appName := args[1]

	if *manifestPath == "" {
		return "", "", "", false, ErrNoManifest
	}

	return appName, *manifestPath, *appPath, *showLogs, nil
}

var (
	ErrNoArgs     = errors.New("app name must be specified")
	ErrNoManifest = errors.New("a manifest is required to push this application")
)

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

func (repo *ApplicationRepo) PushApplication(appName, manifestPath, appPath string, showLogs bool) error {
	args := []string{"push", appName, "-f", manifestPath, "--no-start"}

	if appPath != "" {
		args = append(args, "-p", appPath)
	}

	_, err := repo.conn.CliCommand(args...)
	if err != nil {
		return err
	}

	if showLogs {
		app, err := repo.conn.GetApp(appName)
		if err != nil {
			return err
		}
		dopplerEndpoint, err := repo.conn.DopplerEndpoint()
		if err != nil {
			return err
		}
		token, err := repo.conn.AccessToken()
		if err != nil {
			return err
		}

		cons := consumer.New(dopplerEndpoint, nil, nil)
		defer cons.Close()

		messages, errors := cons.TailingLogs(app.Guid, token)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			for {
				select {
				case m := <-messages:
					if m.GetSourceType() != "STG" { // skip STG messages as the cf tool already prints them
						os.Stderr.WriteString(logs.NewNoaaLogMessage(m).ToLog(time.Local) + "\n")
					}
				case e := <-errors:
					log.Println("error reading logs:", e)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	_, err = repo.conn.CliCommand("start", appName)
	if err != nil {
		return err
	}

	return nil
}

func (repo *ApplicationRepo) DeleteApplication(appName string) error {
	_, err := repo.conn.CliCommand("delete", appName, "-f")
	return err
}

func (repo *ApplicationRepo) ListApplications() error {
	_, err := repo.conn.CliCommand("apps")
	return err
}

func (repo *ApplicationRepo) DoesAppExist(appName string) (bool, error) {
	space, err := repo.conn.GetCurrentSpace()
	if err != nil {
		return false, err
	}

	path := fmt.Sprintf(`v2/apps?q=name:%s&q=space_guid:%s`, url.QueryEscape(appName), space.Guid)
	result, err := repo.conn.CliCommandWithoutTerminalOutput("curl", path)

	if err != nil {
		return false, err
	}

	jsonResp := strings.Join(result, "")

	output := make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonResp), &output)

	if err != nil {
		return false, err
	}

	totalResults, ok := output["total_results"]

	if !ok {
		return false, errors.New("Missing total_results from api response")
	}

	count, ok := totalResults.(float64)

	if !ok {
		return false, fmt.Errorf("total_results didn't have a number %v", totalResults)
	}

	return count == 1, nil
}

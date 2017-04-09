package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/sakkuru/rollback-push/rewind"
)

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		os.Exit(1)
	}
}

func main() {
	plugin.Start(&RollbackPlugin{})
}

type RollbackPlugin struct{}

func g1AppName(appName string) string {
	return versionedAppName(appName, "g1")
}
func g2AppName(appName string) string {
	return versionedAppName(appName, "g2")
}
func versionedAppName(appName string, version string) string {
	return fmt.Sprintf("%s-%s", appName, version)
}
func getActionsForRollback(appRepo *ApplicationRepo, appName string, version string) []rewind.Action {
	return []rewind.Action{
		{
			// start an old app
			Forward: func() error {
				return appRepo.StartApplication(versionedAppName(appName, version))
			},
		},
		{
			Forward: func() error {
				// Map-route it. then unmap the current app
				hostName, err := appRepo.GetHostName(appName)
				if err != nil {
					return fmt.Errorf("Can not get hostname of the %s", appName)
				}
				appRepo.MapRouteApplication(versionedAppName(appName, version), hostName)
				return appRepo.UnMapRouteApplication(appName, hostName)
			},
		},
		{
			Forward: func() error {
				// swap the name between the current app and the old app
				return appRepo.SwapApplication(appName, versionedAppName(appName, version))
			},
		},
	}
}

func getActionsForExistingApp(appRepo *ApplicationRepo, appName, manifestPath, appPath string, g1Exists bool, g2Exists bool) []rewind.Action {
	return []rewind.Action{
		{
			Forward: func() error {
				// versioning
				if g2Exists {
					appRepo.DeleteApplication(g2AppName(appName))
				}
				if g1Exists {
					appRepo.RenameApplication(g1AppName(appName), g2AppName(appName))
				}
				return appRepo.RenameApplication(appName, g1AppName(appName))
			},
		},
		// push
		{
			Forward: func() error {
				return appRepo.PushApplication(appName, manifestPath, appPath)
			},
			ReversePrevious: func() error {
				// If the app cannot start we'll have a lingering application
				// We delete this application so that the rename can succeed
				appRepo.DeleteApplication(appName)

				return appRepo.RenameApplication(g1AppName(appName), appName)
			},
		},
		// unmap-route and stop
		{
			Forward: func() error {
				appRepo.UnMapRouteApplication(g1AppName(appName), appName)
				return appRepo.StopApplication(g1AppName(appName))
			},
		},
	}
}

func getActionsForNewApp(appRepo *ApplicationRepo, appName, manifestPath, appPath string) []rewind.Action {
	return []rewind.Action{
		// push
		{
			Forward: func() error {
				return appRepo.PushApplication(appName, manifestPath, appPath)
			},
		},
	}
}

func (plugin RollbackPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	appRepo := NewApplicationRepo(cliConnection)
	if args[0] == "blue-green-push" {

		appName, manifestPath, appPath, err := ParseArgs(args)
		fatalIf(err)

		appExists, err := appRepo.DoesAppExist(appName)
		fatalIf(err)

		g1Exists, err := appRepo.DoesAppExist(appName + "-g1")
		fatalIf(err)

		g2Exists, err := appRepo.DoesAppExist(appName + "-g2")

		var actionList []rewind.Action

		if appExists {
			actionList = getActionsForExistingApp(appRepo, appName, manifestPath, appPath, g1Exists, g2Exists)
		} else {
			actionList = getActionsForNewApp(appRepo, appName, manifestPath, appPath)
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

	if args[0] == "blue-green-rollback" {
		appName, version, err := ParseRollbackArgs(args)
		fatalIf(err)
		var actionList []rewind.Action

		appExists, err := appRepo.DoesAppExist(appName)
		fatalIf(err)

		if appExists {
			actionList = getActionsForRollback(appRepo, appName, version)

		} else {
			err := fmt.Errorf("Application: %s not found", appName)
			fatalIf(err)
		}
		actions := rewind.Actions{
			Actions:              actionList,
			RewindFailureMessage: "Oh no. Something's gone wrong. I've tried to roll back but you should check to see if everything is OK.",
		}

		err = actions.Execute()
		fatalIf(err)

		fmt.Println()
		fmt.Println(appName + " is swapped with " + versionedAppName(appName, version))
		fmt.Println()
	}

}

func (RollbackPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "rollback",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
			Build: 2,
		},
		Commands: []plugin.Command{
			{
				Name:     "blue-green-push",
				HelpText: "Perform a zero-downtime push with versioning feature of an application over the top of an old one",
				UsageDetails: plugin.Usage{
					Usage: "$ cf blue-green-push application-to-replace \\ \n \t-f path/to/new_manifest.yml \\ \n \t-p path/to/new/path",
				},
			},
			{
				Name:     "blue-green-rollback",
				HelpText: "Perform a rollback from old version",
				UsageDetails: plugin.Usage{
					Usage: "$ cf blue-green-rollback application-name version (e.g. g1 or g2)",
				},
			},
		},
	}
}

func ParseRollbackArgs(args []string) (string, string, error) {
	flags := flag.NewFlagSet("blue-green-rollback", flag.ContinueOnError)
	err := flags.Parse(args[2:])
	if err != nil {
		return "", "", err
	}
	appName := args[1]
	version := args[2]
	return appName, version, nil
}

func ParseArgs(args []string) (string, string, string, error) {
	flags := flag.NewFlagSet("blue-green-push", flag.ContinueOnError)
	manifestPath := flags.String("f", "", "path to an application manifest")
	appPath := flags.String("p", "", "path to application files")

	err := flags.Parse(args[2:])
	if err != nil {
		return "", "", "", err
	}

	appName := args[1]

	if *manifestPath == "" {
		return "", "", "", ErrNoManifest
	}

	return appName, *manifestPath, *appPath, nil
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
func (repo *ApplicationRepo) GetHostName(appName string) (string, error) {
	result, err := repo.conn.GetApp(appName)
	if err != nil {
		return "", err
	}
	return result.Routes[0].Host, err
}

func (repo *ApplicationRepo) GetDomainName(appName string) (string, error) {
	result, err := repo.conn.GetApp(appName)
	if err != nil {
		return "", err
	}
	return result.Routes[0].Domain.Name, err
}

func (repo *ApplicationRepo) MapRouteApplication(appName string, hostName string) error {
	domainName, err := repo.GetDomainName(appName)
	if err != nil {
		return err
	}
	_, err = repo.conn.CliCommand("map-route", appName, domainName, "-n", hostName)
	return err
}

func (repo *ApplicationRepo) UnMapRouteApplication(appName string, hostName string) error {
	domainName, err := repo.GetDomainName(appName)
	// fmt.Println(result)
	if err != nil {
		return err
	}
	_, err = repo.conn.CliCommand("unmap-route", appName, domainName, "-n", hostName)
	return err
}

func (repo *ApplicationRepo) StartApplication(appName string) error {
	_, err := repo.conn.CliCommand("start", appName)
	return err
}

func (repo *ApplicationRepo) StopApplication(appName string) error {
	_, err := repo.conn.CliCommand("stop", appName)
	return err
}

func (repo *ApplicationRepo) RenameApplication(oldName, newName string) error {
	_, err := repo.conn.CliCommand("rename", oldName, newName)
	return err
}
func (repo *ApplicationRepo) SwapApplication(appNameA string, appNameB string) error {
	tempAppName := appNameA + "-now-on-swapping"
	err := repo.RenameApplication(appNameA, tempAppName)
	if err != nil {
		return err
	}
	err = repo.RenameApplication(appNameB, appNameA)
	if err != nil {
		return err
	}
	err = repo.RenameApplication(tempAppName, appNameB)
	return err
}

func (repo *ApplicationRepo) PushApplication(appName, manifestPath, appPath string) error {
	args := []string{"push", appName, "-f", manifestPath}

	if appPath != "" {
		args = append(args, "-p", appPath)
	}

	_, err := repo.conn.CliCommand(args...)
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

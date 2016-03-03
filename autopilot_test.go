package main_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/concourse/autopilot"

	"github.com/cloudfoundry/cli/plugin/fakes"
	plugin_models "github.com/cloudfoundry/cli/plugin/models"
)

func TestAutopilot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Autopilot Suite")
}

var _ = Describe("Flag Parsing", func() {
	It("parses a complete set of args", func() {
		appName, manifestPath, appPath, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname",
				"-f", "manifest-path",
				"-p", "app-path",
			},
		)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(appName).Should(Equal("appname"))
		Ω(manifestPath).Should(Equal("manifest-path"))
		Ω(appPath).Should(Equal("app-path"))
	})

	It("requires a manifest", func() {
		_, _, _, err := ParseArgs(
			[]string{
				"zero-downtime-push",
				"appname",
				"-p", "app-path",
			},
		)
		Ω(err).Should(MatchError(ErrNoManifest))
	})
})

var _ = Describe("ApplicationRepo", func() {
	var (
		cliConn *fakes.FakeCliConnection
		repo    *ApplicationRepo
	)

	BeforeEach(func() {
		cliConn = &fakes.FakeCliConnection{}
		repo = NewApplicationRepo(cliConn)
	})

	Describe("RenameApplication", func() {
		It("renames the application", func() {
			err := repo.RenameApplication("old-name", "new-name")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(cliConn.CliCommandCallCount()).Should(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Ω(args).Should(Equal([]string{"rename", "old-name", "new-name"}))
		})

		It("returns an error if one occurs", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("no app"))

			err := repo.RenameApplication("old-name", "new-name")
			Ω(err).Should(MatchError("no app"))
		})
	})

	Describe("DoesAppExist", func() {

		It("returns an error if the cli returns an error", func() {
			cliConn.CliCommandWithoutTerminalOutputReturns([]string{}, errors.New("you shall not curl"))
			_, err := repo.DoesAppExist("app-name")

			Ω(err).Should(MatchError("you shall not curl"))
		})

		It("returns an error if the cli response is invalid JSON", func() {
			response := []string{
				"}notjson{",
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Ω(err).Should(HaveOccurred())
		})

		It("returns an error if the cli response doesn't contain total_results", func() {
			response := []string{
				`{"brutal_results":2}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Ω(err).Should(MatchError("Missing total_results from api response"))
		})

		It("returns an error if the cli response contains a non-number total_results", func() {
			response := []string{
				`{"total_results":"sandwich"}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			_, err := repo.DoesAppExist("app-name")

			Ω(err).Should(MatchError("total_results didn't have a number sandwich"))
		})

		It("returns true if the app exists", func() {
			response := []string{
				`{"total_results":1}`,
			}
			spaceGUID := "4"

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			cliConn.GetCurrentSpaceReturns(
				plugin_models.Space{
					SpaceFields: plugin_models.SpaceFields{
						Guid: spaceGUID,
					},
				},
				nil,
			)

			result, err := repo.DoesAppExist("app-name")

			Expect(cliConn.CliCommandWithoutTerminalOutputCallCount()).To(Equal(1))
			args := cliConn.CliCommandWithoutTerminalOutputArgsForCall(0)
			Expect(args).To(Equal([]string{"curl", "v2/apps?q=name:app-name&q=space_guid:4"}))

			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).Should(BeTrue())
		})

		It("returns false if the app does not exist", func() {
			response := []string{
				`{"total_results":0}`,
			}

			cliConn.CliCommandWithoutTerminalOutputReturns(response, nil)
			result, err := repo.DoesAppExist("app-name")

			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).Should(BeFalse())
		})

	})

	Describe("PushApplication", func() {
		It("pushes an application with both a manifest and a path", func() {
			err := repo.PushApplication("/path/to/a/manifest.yml", "/path/to/the/app")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(cliConn.CliCommandCallCount()).Should(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Ω(args).Should(Equal([]string{
				"push",
				"-f", "/path/to/a/manifest.yml",
				"-p", "/path/to/the/app",
			}))
		})

		It("pushes an application with only a manifest", func() {
			err := repo.PushApplication("/path/to/a/manifest.yml", "")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(cliConn.CliCommandCallCount()).Should(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Ω(args).Should(Equal([]string{
				"push",
				"-f", "/path/to/a/manifest.yml",
			}))
		})

		It("returns errors from the push", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.PushApplication("/path/to/a/manifest.yml", "/path/to/the/app")
			Ω(err).Should(MatchError("bad app"))
		})
	})

	Describe("DeleteApplication", func() {
		It("deletes all trace of an application", func() {
			err := repo.DeleteApplication("app-name")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(cliConn.CliCommandCallCount()).Should(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Ω(args).Should(Equal([]string{
				"delete", "app-name",
				"-f",
			}))
		})

		It("returns errors from the delete", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad app"))

			err := repo.DeleteApplication("app-name")
			Ω(err).Should(MatchError("bad app"))
		})
	})

	Describe("ListApplications", func() {
		It("lists all the applications", func() {
			err := repo.ListApplications()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(cliConn.CliCommandCallCount()).Should(Equal(1))
			args := cliConn.CliCommandArgsForCall(0)
			Ω(args).Should(Equal([]string{"apps"}))
		})

		It("returns errors from the list", func() {
			cliConn.CliCommandReturns([]string{}, errors.New("bad apps"))

			err := repo.ListApplications()
			Ω(err).Should(MatchError("bad apps"))
		})
	})
})

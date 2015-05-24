package rewind_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/concourse/autopilot/rewind"
)

var _ = Describe("Rewind", func() {
	It("runs through all actions if they're all successful", func() {
		firstRun := false
		secondRun := false

		actions := rewind.Actions{
			Actions: []rewind.Action{
				{
					Forward: func() error {
						firstRun = true
						return nil
					},
				},
				{
					Forward: func() error {
						secondRun = true
						return nil
					},
				},
			},
		}

		err := actions.Execute()
		Ω(err).ShouldNot(HaveOccurred())

		Ω(firstRun).Should(BeTrue())
		Ω(secondRun).Should(BeTrue())
	})

	It("stops and runs the rewind of an action if it fails", func() {
		firstRun := false
		secondRun := false
		secondReverseRun := false
		thirdRun := false

		actions := rewind.Actions{
			Actions: []rewind.Action{
				{
					Forward: func() error {
						firstRun = true
						return nil
					},
				},
				{
					Forward: func() error {
						secondRun = true
						return errors.New("disaster")
					},
					ReversePrevious: func() error {
						secondReverseRun = true
						return nil
					},
				},
				{
					Forward: func() error {
						thirdRun = true
						return nil
					},
				},
			},
		}

		err := actions.Execute()
		Ω(err).Should(MatchError("disaster"))

		Ω(firstRun).Should(BeTrue())
		Ω(secondRun).Should(BeTrue())
		Ω(secondReverseRun).Should(BeTrue())
		Ω(thirdRun).Should(BeFalse())
	})

	It("gives up if the rewind action fails", func() {
		firstRun := false
		secondRun := false
		secondReverseRun := false
		thirdRun := false

		actions := rewind.Actions{
			Actions: []rewind.Action{
				{
					Forward: func() error {
						firstRun = true
						return nil
					},
				},
				{
					Forward: func() error {
						secondRun = true
						return errors.New("disaster")
					},
					ReversePrevious: func() error {
						secondReverseRun = true
						return errors.New("another disaster")
					},
				},
				{
					Forward: func() error {
						thirdRun = true
						return nil
					},
				},
			},
			RewindFailureMessage: "uh oh",
		}

		err := actions.Execute()
		Ω(err).Should(MatchError("uh oh: another disaster"))

		Ω(firstRun).Should(BeTrue())
		Ω(secondRun).Should(BeTrue())
		Ω(secondReverseRun).Should(BeTrue())
		Ω(thirdRun).Should(BeFalse())
	})

	It("just returns the error if a rewind fails with no reverse message", func() {
		firstRun := false
		secondRun := false
		secondReverseRun := false
		thirdRun := false

		actions := rewind.Actions{
			Actions: []rewind.Action{
				{
					Forward: func() error {
						firstRun = true
						return nil
					},
				},
				{
					Forward: func() error {
						secondRun = true
						return errors.New("disaster")
					},
					ReversePrevious: func() error {
						secondReverseRun = true
						return errors.New("another disaster")
					},
				},
				{
					Forward: func() error {
						thirdRun = true
						return nil
					},
				},
			},
		}

		err := actions.Execute()
		Ω(err).Should(MatchError("another disaster"))

		Ω(firstRun).Should(BeTrue())
		Ω(secondRun).Should(BeTrue())
		Ω(secondReverseRun).Should(BeTrue())
		Ω(thirdRun).Should(BeFalse())
	})
})

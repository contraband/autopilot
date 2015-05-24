package rewind

import "fmt"

type Actions struct {
	Actions []Action

	RewindFailureMessage string
}

func (actions Actions) Execute() error {
	for _, action := range actions.Actions {
		err := action.Forward()
		if err != nil {
			if action.ReversePrevious == nil {
				return err
			}

			reverseError := action.ReversePrevious()
			if reverseError != nil {
				if actions.RewindFailureMessage != "" {
					return fmt.Errorf("%s: %s", actions.RewindFailureMessage, reverseError)
				} else {
					return reverseError
				}
			}

			return err
		}
	}

	return nil
}

type Action struct {
	Forward         func() error
	ReversePrevious func() error
}

package workflow

import "context"

// Phase is a single phase of the workflow.
type Phase struct {
	// Name is the name of the phase
	Name string

	// Description is the description of the phase
	Description string

	// PreRun is the function to run before the phase
	PreRun func(data RunData) error

	// Run is the function to run the phase
	Run func(ctx context.Context, data RunData) error
}

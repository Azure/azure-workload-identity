package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// RunData contains the data that is passed to the phases
type RunData = interface{}

// Runner is the interface for running phases
type Runner interface {
	AppendPhases(phases ...Phase)
	BindToCommand(cmd *cobra.Command)
	Run(data RunData) error
}

// runner is the default implementation of the Runner interface
type runner struct {
	skipPhases []string
	phases     []Phase
}

var _ Runner = &runner{}

// NewRunner returns a new instance of the runner
func NewPhaseRunner() Runner {
	return &runner{}
}

// AppendPhases adds a phase to the list of phases to run
func (r *runner) AppendPhases(phases ...Phase) {
	r.phases = append(r.phases, phases...)
}

// BindToCommand alters the command's help text and flags to include the phase's flags
func (r *runner) BindToCommand(cmd *cobra.Command) {
	// Alter the command's help text
	if cmd.Short == "" {
		cmd.Short = fmt.Sprintf("%s a workload identity", cmd.Use)
	}
	if cmd.Long == "" {
		long := fmt.Sprintf("The \"%s\" command executes the following phases in order:", cmd.Use)

		// Add extra padding to align the phase names
		longest := 0
		for _, phase := range r.phases {
			if longest < len(phase.Name) {
				longest = len(phase.Name)
			}
		}
		for _, phase := range r.phases {
			paddingCount := longest - len(phase.Name)
			long += fmt.Sprintf("\n%s%s  %s", phase.Name, strings.Repeat(" ", paddingCount), phase.Description)
		}
		cmd.Long = long
	}

	// Common flags between commands
	cmd.Flags().StringSliceVar(&r.skipPhases, "skip-phases", []string{}, "List of phases to skip")
}

// Run runs the phases except the ones specified in skipPhases
func (r *runner) Run(data RunData) error {
	skipPhases, err := r.computeSkipPhases()
	if err != nil {
		return errors.Wrap(err, "failed to compute skip phases")
	}

	filtered := []Phase{}
	for _, phase := range r.phases {
		if skipPhases[phase.Name] {
			log.WithField("phase", phase.Name).Info("skipping phase")
			continue
		}
		filtered = append(filtered, phase)
	}

	// Run PreRun for all phases before executing the phases
	for _, phase := range filtered {
		if err := phase.PreRun(data); err != nil {
			return errors.Wrapf(err, "failed to run pre-run for phase %s", phase.Name)
		}
	}

	for _, phase := range filtered {
		if err := phase.Run(context.Background(), data); err != nil {
			return errors.Wrapf(err, "failed to run phase %s", phase.Name)
		}
	}

	return nil
}

// computeSkipPhases computes the list of phases to skip based on the skip-phases flag
func (r *runner) computeSkipPhases() (map[string]bool, error) {
	currentPhases := make(map[string]bool)
	for _, phase := range r.phases {
		currentPhases[phase.Name] = true
	}

	skipPhases := make(map[string]bool)
	for _, p := range r.skipPhases {
		// check if the phases specified in skip-phases are valid
		if _, ok := currentPhases[p]; !ok {
			validPhases := make([]string, 0, len(currentPhases))
			for _, pp := range r.phases {
				validPhases = append(validPhases, pp.Name)
			}
			return nil, errors.Errorf("phase '%s' not found. Valid phases are: %v", p, validPhases)
		}
		skipPhases[p] = true
	}

	return skipPhases, nil
}

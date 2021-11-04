package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// RunData contains the data that is passed to the phases
type RunData = interface{}

// Runner is the interface for running phases
type Runner interface {
	// AppendPhases adds a phase to the list of phases to run
	AppendPhases(phases ...Phase)

	// BindToCommand alters the command's help text and flags to include the phase's flags
	BindToCommand(cmd *cobra.Command, data RunData)

	// Run runs the phases except the ones specified in skipPhases
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
func (r *runner) BindToCommand(cmd *cobra.Command, data RunData) {
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

	// common flags between commands
	cmd.Flags().StringSliceVar(&r.skipPhases, "skip-phases", []string{}, "List of phases to skip")

	// add the phase command, enabling the user to specify the phase to run
	phaseCmd := &cobra.Command{
		Use:   "phase",
		Short: fmt.Sprintf("The \"phase\" command invokes a single phase of the %s workflow", cmd.Use),
	}
	for _, phase := range r.phases {
		// workaround: create a copy of the variable 'phase' so that each subcommand
		// gets its own 'phase' variable instead of sharing the iterator variable
		p := phase

		subcommand := &cobra.Command{
			Use:     p.Name,
			Aliases: p.Aliases,
			Short:   p.Description,
			RunE: func(c *cobra.Command, args []string) error {
				// only run this particular phase
				r.phases = []Phase{p}
				return r.Run(data)
			},
		}
		inheritsFlags(cmd.Flags(), subcommand.Flags(), p.Flags)
		phaseCmd.AddCommand(subcommand)
	}

	cmd.AddCommand(phaseCmd)
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

// inheritFlags copies flags from the parent command to the child command.
// xref: https://github.com/kubernetes/kubernetes/blob/1f9d448283a7915df9d617708468f06ba17aaaa7/cmd/kubeadm/app/cmd/phases/workflow/runner.go#L400-L414
func inheritsFlags(sourceFlags, targetFlags *pflag.FlagSet, cmdFlags []string) {
	// If the list of flag to be inherited from the parent command is not defined, no flag is added
	if cmdFlags == nil {
		return
	}

	// add all the flags to be inherited to the target flagSet
	sourceFlags.VisitAll(func(f *pflag.Flag) {
		for _, c := range cmdFlags {
			if f.Name == c {
				targetFlags.AddFlag(f)
			}
		}
	})
}

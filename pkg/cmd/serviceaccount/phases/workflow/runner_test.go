package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func TestAppendPhases(t *testing.T) {
	r := &runner{}
	r.AppendPhases(Phase{
		Name: "phase-1",
	}, Phase{
		Name: "phase-2",
	})
	if len(r.phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(r.phases))
	}
	for i, phase := range r.phases {
		if phase.Name != "phase-"+fmt.Sprintf("%d", i+1) {
			t.Errorf("expected phase-%d to be named %q, got %q", i+1, "phase-"+fmt.Sprintf("%d", i+1), phase.Name)
		}
	}
}

func TestAppendSkipPhases(t *testing.T) {
	r := &runner{
		skipPhases: []string{"phase-1"},
	}
	r.AppendSkipPhases(Phase{
		Name: "phase-2",
	}, Phase{
		Name: "phase-3",
	})
	if len(r.skipPhases) != 3 {
		t.Errorf("expected 2 phases, got %d", len(r.skipPhases))
	}
	for i, phase := range r.skipPhases {
		if phase != "phase-"+fmt.Sprintf("%d", i+1) {
			t.Errorf("expected phase-%d to be named %q, got %q", i+1, "phase-"+fmt.Sprintf("%d", i+1), phase)
		}
	}
}

func TestIsPhaseActive(t *testing.T) {
	tests := []struct {
		name       string
		skipPhases []string
		phase      string
		expect     bool
	}{
		{
			name:       "no skip phase",
			skipPhases: []string{},
			phase:      "phase-1",
			expect:     true,
		},
		{
			name:       "skip phase",
			skipPhases: []string{"phase-1"},
			phase:      "phase-1",
			expect:     false,
		},
		{
			name:       "skip phase",
			skipPhases: []string{"phase-1"},
			phase:      "phase-2",
			expect:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &runner{
				skipPhases: test.skipPhases,
			}
			if r.IsPhaseActive(Phase{Name: test.phase}) != test.expect {
				t.Errorf("expected IsPhaseActive to return %v, got %v", test.expect, r.IsPhaseActive(Phase{Name: test.phase}))
			}
		})
	}
}

func TestRun(t *testing.T) {
	order := 1
	r := &runner{
		phases: []Phase{
			{
				Name: "phase-1",
				PreRun: func(data RunData) error {
					if order != 1 {
						return errors.Errorf("expected order to be %d, got %d", 1, order)
					}
					order++
					return nil
				},
				Run: func(ctx context.Context, data RunData) error {
					if order != 3 {
						return errors.Errorf("expected order to be %d, got %d", 2, order)
					}
					order++
					return nil
				},
			},
			{
				Name: "phase-2",
				PreRun: func(data RunData) error {
					if order != 2 {
						return errors.Errorf("expected order to be %d, got %d", 1, order)
					}
					order++
					return nil
				},
				Run: func(ctx context.Context, data RunData) error {
					if order != 4 {
						return errors.Errorf("expected order to be %d, got %d", 2, order)
					}
					order++
					return nil
				},
			},
			{
				Name: "phase-3",
				PreRun: func(data RunData) error {
					return errors.Errorf("expected phase-3 to be skipped")
				},
				Run: func(ctx context.Context, data RunData) error {
					return errors.Errorf("expected phase-3 to be skipped")
				},
			},
		},
		skipPhases: []string{"phase-3"},
	}

	if err := r.Run(nil); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBindToCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}
	r := &runner{}
	r.AppendPhases(Phase{
		Name:        "phase-1",
		Description: "phase-1 description and some random string",
	}, Phase{
		Name:        "phase-2 plus some random string",
		Description: "phase-2 description plus some random string on top of some other random string",
	})

	r.BindToCommand(cmd, nil)
	if cmd.Short != "test a workload identity" {
		t.Errorf("expected short description to be %q, got %q", "test a workload identity", cmd.Short)
	}
	expectedLong := "The \"test\" command executes the following phases in order:\nphase-1                          phase-1 description and some random string\nphase-2 plus some random string  phase-2 description plus some random string on top of some other random string"
	if cmd.Long != expectedLong {
		t.Errorf("expected long description to be %q, got %q", expectedLong, cmd.Long)
	}

	if cmd.Flag("skip-phases") == nil {
		t.Errorf("expected --skip-phases flag to be added")
	}
}

func TestComputeSkipPhases(t *testing.T) {
	tests := []struct {
		name       string
		skipPhases []string
		expect     map[string]bool
		errorMsg   string
	}{
		{
			name:       "no skip phase",
			skipPhases: []string{},
			expect:     map[string]bool{},
			errorMsg:   "",
		},
		{
			name:       "skip one phase",
			skipPhases: []string{"phase-1"},
			expect: map[string]bool{
				"phase-1": true,
			},
			errorMsg: "",
		},
		{
			name:       "skip unknown phases",
			skipPhases: []string{"phase-1", "phase-2", "phase-3"},
			expect:     map[string]bool{},
			errorMsg:   "phase 'phase-3' not found. Valid phases are: [phase-1 phase-2]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &runner{
				skipPhases: test.skipPhases,
			}
			r.AppendPhases(Phase{
				Name: "phase-1",
			}, Phase{
				Name: "phase-2",
			})

			skip, err := r.computeSkipPhases()
			if err != nil {
				if test.errorMsg == "" {
					t.Errorf("expected no error, got %v", err)
				} else if err.Error() != test.errorMsg {
					t.Errorf("expected error message to be %q, got %q", test.errorMsg, err.Error())
				}
			} else {
				if test.errorMsg != "" {
					t.Errorf("expected error message to be %q, got no error", test.errorMsg)
				}
				if len(skip) != len(test.expect) {
					t.Errorf("expected %d phases to be skipped, got %d", len(test.expect), len(skip))
				}
				for name, expect := range test.expect {
					if skip[name] != expect {
						t.Errorf("expected phase %q to be skipped, got skipped=%t", name, skip[name])
					}
				}
			}
		})
	}
}

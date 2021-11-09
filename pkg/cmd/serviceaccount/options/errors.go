package options

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// FlagIsRequiredError is returned when a required flag is not set
func FlagIsRequiredError(name string) error {
	return errors.Errorf("--%s is required", name)
}

// OneOfFlagsIsRequiredError is returned when at least one of the flags is required
func OneOfFlagsIsRequiredError(names ...string) error {
	flags := fmt.Sprintf("--%s", strings.Join(names, " or --"))
	return errors.Errorf("%s is required", flags)
}

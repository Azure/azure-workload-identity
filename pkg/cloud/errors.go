package cloud

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
)

// IsNotFound returns true if the given error is a NotFound error.
func IsNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

// IsRoleAssignmentAlreadyDeleted returns true if the given error is a role assignment already deleted error.
// Ref: https://docs.microsoft.com/en-us/rest/api/authorization/role-assignments/delete#response
func IsRoleAssignmentAlreadyDeleted(err error) bool {
	derr := autorest.DetailedError{}
	return errors.As(err, &derr) && derr.StatusCode == http.StatusNoContent
}

// IsResourceNotFound parses the error message to check if it's resource not found error.
func IsResourceNotFound(err error) bool {
	derr := autorest.DetailedError{}
	return errors.As(err, &derr) && derr.StatusCode == http.StatusNotFound
}

// IsAlreadyExists parses the error message to check if it's resource already exists error.
func IsAlreadyExists(err error) bool {
	derr := autorest.DetailedError{}
	return errors.As(err, &derr) && derr.StatusCode == http.StatusConflict
}

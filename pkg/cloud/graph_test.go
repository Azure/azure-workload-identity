package cloud

import "testing"

func TestGetDisplayNameFilter(t *testing.T) {
	got := getDisplayNameFilter("test")
	want := "displayName eq 'test'"

	if got != want {
		t.Errorf("getDisplayNameFilter() = %v, want %v", got, want)
	}
}

func TestGetSubjectFilter(t *testing.T) {
	got := getSubjectFilter("test")
	want := "subject eq 'test'"

	if got != want {
		t.Errorf("getSubjectFilter() = %v, want %v", got, want)
	}
}

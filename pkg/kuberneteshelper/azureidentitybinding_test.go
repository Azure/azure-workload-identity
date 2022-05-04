package kuberneteshelper

import (
	"reflect"
	"sort"
	"testing"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSort(t *testing.T) {
	slice := []aadpodv1.AzureIdentityBinding{{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "test",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test3",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "test",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "default",
		},
	}}
	expected := []aadpodv1.AzureIdentityBinding{{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test3",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "test",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "test",
		},
	}}
	sort.Sort(azureIdentityBindings(slice))
	if !reflect.DeepEqual(slice, expected) {
		t.Errorf("expected %v, got %v", expected, slice)
	}
}

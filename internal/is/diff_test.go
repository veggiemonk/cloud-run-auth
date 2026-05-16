package is

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDiffEqualValuesReturnsEmpty(t *testing.T) {
	got := []string{"a", "b"}
	want := []string{"a", "b"}

	if diff := Diff(got, want); diff != "" {
		t.Fatalf("Diff() = %q, want empty", diff)
	}
}

func TestDiffNilAndEmptySliceDifferByDefault(t *testing.T) {
	var got []string
	want := []string{}

	if diff := Diff(got, want); diff == "" {
		t.Fatal("Diff() = empty, want non-empty diff")
	}
}

func TestDiffNilAndEmptySliceCanBeEquatedExplicitly(t *testing.T) {
	var got []string
	want := []string{}

	if diff := Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
		t.Fatalf("Diff() = %q, want empty", diff)
	}
}

func TestDiffComparesUnexportedFields(t *testing.T) {
	type thing struct {
		hidden string
	}

	got := thing{hidden: "a"}
	want := thing{hidden: "b"}

	if diff := Diff(got, want); diff == "" {
		t.Fatal("Diff() = empty, want non-empty diff")
	}
}

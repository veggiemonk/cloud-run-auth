package is

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEqualPassesCmpOptionsThrough(t *testing.T) {
	var got []string
	want := []string{}

	Equal(t, got, want, "", cmpopts.EquateEmpty())
}

func TestEqualPrintsMessageOnFailure(t *testing.T) {
	out := runEqualFailureTest(t, "TestEqualFailureWithMessage")
	if !strings.Contains(out, "numbers differ") {
		t.Fatalf("output = %q, want contains message", out)
	}
}

func TestEqualSuppressesEmptyMessage(t *testing.T) {
	out := runEqualFailureTest(t, "TestEqualFailureWithoutMessage")
	if strings.Contains(out, "\n\n-got +want") {
		t.Fatalf("output = %q, want no blank message line before diff", out)
	}
}

func TestEqualFailureWithMessage(t *testing.T) {
	if os.Getenv("IS_EQUAL_FAIL") == "" {
		t.Skip("helper test")
	}

	Equal(t, 1, 2, "numbers differ")
}

func TestEqualFailureWithoutMessage(t *testing.T) {
	if os.Getenv("IS_EQUAL_FAIL") == "" {
		t.Skip("helper test")
	}

	Equal(t, 1, 2, "")
}

func runEqualFailureTest(t *testing.T, testName string) string {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "go", "test", ".", "-run", "^"+testName+"$", "-count=1")
	cmd.Env = append(os.Environ(), "IS_EQUAL_FAIL=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go test succeeded unexpectedly: %s", out)
	}

	return string(out)
}

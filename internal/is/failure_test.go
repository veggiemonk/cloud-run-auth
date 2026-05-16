package is

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestFailureIncludesSingleCallerFrame(t *testing.T) {
	out := runFailureCase(t, "TestFailureCaseEqual")
	if !strings.Contains(out, "failure_test.go:") {
		t.Fatalf("output = %q, want caller frame", out)
	}
}

func TestFailureIncludesCallerChain(t *testing.T) {
	out := runFailureCase(t, "TestFailureCaseNestedEqual")
	if !strings.Contains(out, "failure_test.go:") || !strings.Contains(out, " -> ") {
		t.Fatalf("output = %q, want caller chain", out)
	}
}

func TestFailureOmitsBlankMessageLine(t *testing.T) {
	out := runFailureCase(t, "TestFailureCaseEqualNoMessage")
	if strings.Contains(out, "\n\n-got +want") {
		t.Fatalf("output = %q, want no blank message line", out)
	}
}

func TestTrueOutputStable(t *testing.T) {
	out := runFailureCase(t, "TestFailureCaseTrue")
	if !strings.Contains(out, "not true") {
		t.Fatalf("output = %q, want 'not true'", out)
	}
}

func TestNoErrOutputStable(t *testing.T) {
	out := runFailureCase(t, "TestFailureCaseNoErr")
	if !strings.Contains(out, "boom") {
		t.Fatalf("output = %q, want error text", out)
	}
}

func TestCallersNoPanicOnShallowStack(t *testing.T) {
	callers := Callers()
	if callers == "" {
		t.Fatal("Callers() = empty, want caller frame")
	}
}

func TestFailureCaseEqual(t *testing.T) {
	if os.Getenv("IS_FAILURE_CASE") == "" {
		t.Skip("helper test")
	}

	Equal(t, 1, 2, "numbers differ")
}

func TestFailureCaseEqualNoMessage(t *testing.T) {
	if os.Getenv("IS_FAILURE_CASE") == "" {
		t.Skip("helper test")
	}

	Equal(t, 1, 2, "")
}

func TestFailureCaseNestedEqual(t *testing.T) {
	if os.Getenv("IS_FAILURE_CASE") == "" {
		t.Skip("helper test")
	}

	nestedEqual(t)
}

func nestedEqual(t *testing.T) {
	t.Helper()
	deeperEqual(t)
}

func deeperEqual(t *testing.T) {
	t.Helper()
	Equal(t, 1, 2, "nested numbers differ")
}

func TestFailureCaseTrue(t *testing.T) {
	if os.Getenv("IS_FAILURE_CASE") == "" {
		t.Skip("helper test")
	}

	True(t, false)
}

func TestFailureCaseNoErr(t *testing.T) {
	if os.Getenv("IS_FAILURE_CASE") == "" {
		t.Skip("helper test")
	}

	NoErr(t, errors.New("boom"))
}

func runFailureCase(t *testing.T, testName string) string {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "go", "test", ".", "-run", "^"+testName+"$", "-count=1")
	cmd.Env = append(os.Environ(), "IS_FAILURE_CASE=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go test succeeded unexpectedly: %s", out)
	}

	return string(out)
}

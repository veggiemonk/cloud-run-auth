//go:build mage

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Help

const (
	binDir  = "bin"
	distDir = "dist"
)

var (
	releaseOSes   = []string{"linux", "darwin"}
	releaseArches = []string{"amd64", "arm64"}
	goFlags       = "-trimpath"
	ldFlags       = "-s -w"
)

// Each entry maps a binary name to its main package.
var binaries = map[string]string{
	"runiap":       "./cmd/runiap",
	"runoauth":     "./cmd/runoauth",
	"runoauthprod": "./cmd/runoauthprod",
}

func runCmd(cmd string, args ...string) error {
	return sh.RunV(cmd, args...)
}

// Help shows available commands (same as go tool mage -l)
func Help() {
	runCmd("go", "tool", "mage", "-l")
}

// Full verify gate (what CI runs; identical locally)
func Ci() {
	mg.Deps(Check)
}

// Generate + tidy + lint + vuln + test + build
func Check() {
	mg.SerialDeps(Generate, Tidy, Lint, Vuln, Test, Build)
}

// Generate templ Go code from .templ files
func Generate() error {
	return runCmd("go", "tool", "templ", "generate")
}

// Build all binaries into bin/
func Build() error {
	mg.Deps(Generate)
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	for name, pkg := range binaries {
		out := filepath.Join(binDir, name)
		if err := runCmd("go", "build", goFlags, "-ldflags", ldFlags, "-o", out, pkg); err != nil {
			return err
		}
	}
	return nil
}

// go install all binaries
func Install() error {
	mg.Deps(Generate)
	for _, pkg := range binaries {
		if err := runCmd("go", "install", goFlags, "-ldflags", ldFlags, pkg); err != nil {
			return err
		}
	}
	return nil
}

// Run tests with race
func Test() error {
	return runCmd("go", "test", "-race", "-count=1", "-shuffle=on", "-timeout=10m", "./...")
}

// Run tests verbose
func TestV() error {
	return runCmd("go", "test", "-race", "-count=1", "-shuffle=on", "-timeout=10m", "-v", "./...")
}

// Coverage report
func Cover() error {
	defer os.Remove("coverage.out")
	if err := runCmd("go", "test", "-race", "-count=1", "-shuffle=on", "-timeout=10m", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	return runCmd("go", "tool", "cover", "-func=coverage.out")
}

// Benchmarks
func Bench() error {
	return runCmd("go", "test", "-run=XXX", "-bench=.", "-benchmem", "./...")
}

// Run linters (go vet + golangci-lint)
func Lint() error {
	mg.Deps(Vet)
	return runCmd("golangci-lint", "run")
}

// go vet
func Vet() error {
	return runCmd("go", "vet", "./...")
}

// govulncheck via go tool
func Vuln() error {
	return runCmd("go", "tool", "govulncheck", "./...")
}

// Tidy + go fix + modernize + lint --fix + fieldalignment + formatters + templ fmt
func Fix() {
	mg.Deps(CheckTools)
	runCmd("go", "mod", "tidy")
	runCmd("go", "fix", "./...")
	Modernize()
	runCmd("golangci-lint", "run", "--fix")
	FixAlign()
	runCmd("gofumpt", "-w", ".")
	runCmd("goimports", "-w", ".")
	runCmd("go", "tool", "templ", "fmt", ".")
}

// Apply modernize analyzer fixes
func Modernize() error {
	return runCmd("go", "tool", "modernize", "-fix", "./...")
}

// Iterate fieldalignment -fix until stable
func FixAlign() {
	for i := 0; i < 5; i++ {
		_ = sh.Run("go", "tool", "fieldalignment", "-fix", "./...")
	}
}

// go mod tidy
func Tidy() error {
	return runCmd("go", "mod", "tidy")
}

// Bump all deps, then tidy
func Update() error {
	mg.Deps(UpdateActions)
	if err := runCmd("go", "get", "-u", "go@latest"); err != nil {
		return err
	}
	if err := runCmd("go", "get", "-u", "./..."); err != nil {
		return err
	}
	return runCmd("go", "mod", "tidy")
}

// Refresh pinned GH Action SHAs via ratchet
func UpdateActions() error {
	files, err := filepath.Glob(".github/workflows/*.yml")
	if err != nil || len(files) == 0 {
		fmt.Println("no .github/workflows/*.yml; nothing to do")
		return nil
	}
	args := append([]string{"upgrade"}, files...)
	return runCmd("ratchet", args...)
}

// Inline embedmd directives in markdown files
func Doc() error {
	var mdFiles []string
	_ = filepath.Walk("docs", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})
	args := []string{"tool", "embedmd", "-w", "README.md"}
	args = append(args, mdFiles...)
	return runCmd("go", args...)
}

// Build container images for all binaries with ko
func Ko() error {
	mg.Deps(Generate)
	for name := range binaries {
		if err := runCmd("ko", "build", binaries[name], "--bare", "--platform=linux/amd64"); err != nil {
			return err
		}
	}
	return nil
}

// Test goreleaser locally (no publish)
func ReleaseSnapshot() error {
	return runCmd("goreleaser", "release", "--snapshot", "--clean")
}

// Verify required host tools are on PATH
func CheckTools() error {
	missing := false
	tools := []string{"golangci-lint", "ko", "goreleaser"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = true
			fmt.Printf("ERROR: '%s' not found on PATH.\n", tool)
		}
	}

	if missing {
		gobin := os.Getenv("GOBIN")
		if gobin == "" {
			out, _ := sh.Output("go", "env", "GOPATH")
			gobin = filepath.Join(out, "bin")
		}
		fmt.Printf("  GOBIN resolves to: %s\n", gobin)
		fmt.Printf("  Your PATH:         %s\n", os.Getenv("PATH"))
		fmt.Println("  Fix: install the missing host tools and ensure $GOBIN is on PATH.")
		return fmt.Errorf("missing required tools")
	}
	return nil
}

// Cross-compile every binary to dist/
func Release() error {
	mg.Deps(Generate)
	os.RemoveAll(distDir)
	os.MkdirAll(distDir, 0o755)

	for binary, pkg := range binaries {
		for _, osName := range releaseOSes {
			for _, arch := range releaseArches {
				ext := ""
				if osName == "windows" {
					ext = ".exe"
				}
				out := filepath.Join(distDir, fmt.Sprintf("%s_%s_%s%s", binary, osName, arch, ext))
				fmt.Printf("→ %s\n", out)

				env := map[string]string{
					"GOOS":   osName,
					"GOARCH": arch,
				}
				if err := sh.RunWithV(env, "go", "build", goFlags, "-ldflags", ldFlags, "-o", out, pkg); err != nil {
					return err
				}
			}
		}
	}

	sumsFile, err := os.Create(filepath.Join(distDir, "SHA256SUMS"))
	if err != nil {
		return err
	}
	defer sumsFile.Close()

	files, _ := os.ReadDir(distDir)
	for _, f := range files {
		if f.Name() == "SHA256SUMS" || f.IsDir() {
			continue
		}
		path := filepath.Join(distDir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		hash := sha256.Sum256(data)
		fmt.Fprintf(sumsFile, "%s  %s\n", hex.EncodeToString(hash[:]), f.Name())
	}
	fmt.Printf("→ %s/SHA256SUMS\n", distDir)

	return nil
}

// Build then run a binary by name: mage run runiap
func Run(name string) error {
	if _, ok := binaries[name]; !ok {
		names := make([]string, 0, len(binaries))
		for n := range binaries {
			names = append(names, n)
		}
		return fmt.Errorf("unknown binary %q; known: %s", name, strings.Join(names, ", "))
	}
	mg.Deps(Build)
	return runCmd("./" + filepath.Join(binDir, name))
}

// Install the repo's git pre-commit hook
func Hooks() error {
	content, err := os.ReadFile("scripts/pre-commit")
	if err != nil {
		return err
	}
	if err := os.WriteFile(".git/hooks/pre-commit", content, 0o755); err != nil {
		return err
	}
	fmt.Println("installed .git/hooks/pre-commit")
	return nil
}

// Remove build artefacts
func Clean() {
	os.RemoveAll(binDir)
	os.RemoveAll(distDir)
	os.Remove("coverage.out")
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	Dockerfile                = "Dockerfile"
	GitHubActionsWorkflowsDir = ".github/workflows"
)

// syncgover is a tool to read the desired Golang version from the Dockerfile `go` image version tag,
// and update the `go` version in the `go.mod` file and GitHub Actions workflow file(s) accordingly.
//
// This command is intended to be run in a GitHub Actions workflow step before `go test` and image-building steps,
// so that changes made by this tool can be tested and the test results are reflected in the status check.
func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := syncGoVer(dir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// syncGoVer reads the desired Golang version from the Dockerfile `go` image version tag,
// and updates the `go` version in the `go.mod` file and GitHub Actions workflow file(s) accordingly.
func syncGoVer(wd string) error {
	var (
		dockerfile                = filepath.Join(wd, Dockerfile)
		gitHubActionsWorkflowsDir = filepath.Join(wd, GitHubActionsWorkflowsDir)
	)

	content, err := os.ReadFile(dockerfile)
	if err != nil {
		return fmt.Errorf("could not read the Dockerfile: %w", err)
	}

	goVersion, err := readGoVersionFromDockerfile(string(content))
	if err != nil {
		return fmt.Errorf("could not read the Go version from the Dockerfile: %w", err)
	}

	major := strings.Split(goVersion, ".")[0]
	minor := strings.Split(goVersion, ".")[1]
	minorInt, err := strconv.Atoi(minor)
	if err != nil {
		return fmt.Errorf("could not convert the minor version to an integer: %w", err)
	}
	oneMinusMinor := fmt.Sprintf("%s.%d", major, minorInt-1)

	// Update the `go` version in the `go.mod` file.
	// Note that we don't pass `"-go", goVersion`, because it can only set up to the version of the go command
	// used to build this tool, which should be one minor or patch version older than the "next" version in the Dockerfile.
	if err := runCommand(wd, "go", "mod", "tidy", "-go", oneMinusMinor); err != nil {
		return fmt.Errorf("could not update the `go` version in the `go.mod` file: %w", err)
	}

	// Update the `go` version in the GitHub Actions workflow file(s).
	if err := filepath.Walk(gitHubActionsWorkflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("could not walk the GitHub Actions workflow directory: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		// Update the `go` version in the GitHub Actions workflow file.
		if err := replaceGoVersioninWorkflow(path, goVersion); err != nil {
			return fmt.Errorf("could not update the `go` version in the GitHub Actions workflow file: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	// Exit with 0 if there are no changes to commit.
	if err := runCommand(wd, "git", "diff", "--exit-code"); err == nil {
		return nil
	}

	// git add/commit/push the changes
	if err := runCommand(wd, "git", "add", "-u"); err != nil {
		return fmt.Errorf("could not git add the changes: %w", err)
	}

	if err := runCommand(wd, "git", "commit", "-m", "Update the `go` version in the `go.mod` file and GitHub Actions workflow file(s)"); err != nil {
		return fmt.Errorf("could not git commit the changes: %w", err)
	}

	return nil
}

func readGoVersionFromDockerfile(content string) (string, error) {
	// Look for the `FROM golang:<version>` line in the Dockerfile,
	// and return the version number.

	buf := strings.NewReader(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "FROM golang:") {
			return strings.TrimPrefix(line, "FROM golang:"), nil
		}
	}

	return "", fmt.Errorf("could not find the `FROM golang:<version>` line in the Dockerfile")
}

func runCommand(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %s %s failed: %s: %w", command, strings.Join(args, " "), string(out), err)
	}

	return nil
}

func replaceGoVersioninWorkflow(file string, goVersion string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("could not read the GitHub Actions workflow file: %w", err)
	}

	var newContent strings.Builder

	buf := strings.NewReader(string(content))
	scanner := bufio.NewScanner(buf)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "go-version: ") {
			splits := strings.Split(line, ": ")
			if len(splits) != 2 {
				return fmt.Errorf("could not split the `go-version` line: %s", line)
			}

			newContent.WriteString(splits[0])
			newContent.WriteString(": ")
			newContent.WriteString(goVersion)
			newContent.WriteString("\n")
			continue
		}

		newContent.WriteString(fmt.Sprintf("%s\n", line))
	}

	if err := os.WriteFile(file, []byte(newContent.String()), 0644); err != nil {
		return fmt.Errorf("could not write the GitHub Actions workflow file: %w", err)
	}

	return nil
}

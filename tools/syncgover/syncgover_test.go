package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncgover(t *testing.T) {
	localRepoPath := filepath.Join(t.TempDir(), "local-repo")

	require.NoError(t, os.MkdirAll(localRepoPath, 0755))

	recursiveCopy(t, filepath.Join("testdata", "input"), localRepoPath)
	gitInitAndCommit(t, localRepoPath)

	require.NoError(t, syncGoVer(localRepoPath))

	// We compare the output against the snapshot,
	// so that Dockerfile is unchanged, while GitHub Actions workflow files are updated.
	compareAgainstSnapshot(t, localRepoPath, filepath.Join("testdata", "output"))
	// We also ensure that the `go` version in the `go.mod` file is updated.
	// This is not covered by the snapshot comparison, because `go mod tidy`
	// may change the `go.mod` file in an unpredictable way.
	// Example: https://github.com/golang/go/issues/65847
	ensureGoModGoVersion(t, localRepoPath, "1.22")
}

func recursiveCopy(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		return copyFile(path, dstPath)
	})

	require.NoError(t, err)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func gitInitAndCommit(t *testing.T, localRepoPath string) {
	t.Helper()
	require.NoError(t, runCommand(localRepoPath, "git", "init"), "git init")
	require.NoError(t, runCommand(localRepoPath, "git", "add", "."), "git add .")
	require.NoError(t, runCommand(localRepoPath, "git", "commit", "-m", "Initial commit"), "git commit")
}

func compareAgainstSnapshot(t *testing.T, got, want string) {
	t.Helper()

	require.NoError(t, filepath.Walk(want, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(want, path)
		if err != nil {
			return err
		}

		gotPath := filepath.Join(got, relPath)
		if info.IsDir() {
			return nil
		}

		wantContent, err := os.ReadFile(path)
		require.NoError(t, err)

		gotContent, err := os.ReadFile(gotPath)
		require.NoError(t, err)

		require.Equal(t, string(wantContent), string(gotContent))
		return nil
	}))
}

func ensureGoModGoVersion(t *testing.T, localRepoPath, want string) {
	goModFile := filepath.Join(localRepoPath, "go.mod")
	goModFileContent, err := os.ReadFile(goModFile)
	require.NoError(t, err)
	goVersion, err := readGoVersionFromGoMod(string(goModFileContent))
	require.NoError(t, err)
	require.Equal(t, "1.22", goVersion)
}

func readGoVersionFromGoMod(content string) (string, error) {
	// Look for the `go` version in the `go.mod` file,
	// and return the version number.

	buf := strings.NewReader(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "go ") {
			return strings.TrimPrefix(line, "go "), nil
		}
	}

	return "", fmt.Errorf("could not find the `go` version in the `go.mod` file")
}

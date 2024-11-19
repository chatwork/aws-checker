package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGover(t *testing.T) {
	goDevDLJson := filepath.Join("testdata", "go.dev.dl.json")

	data, err := os.ReadFile(goDevDLJson)
	require.NoError(t, err)

	versions, err := getGoVersions(data)
	require.NoError(t, err)

	require.Equal(t, [2]string{"1.23.3", "1.22.9"}, versions)
}

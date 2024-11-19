package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchGoVer(t *testing.T) {
	goDevDLJson, err := os.ReadFile(filepath.Join("testdata", "go.dev.dl.json"))
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, setGHAOutputs(&b, func(string) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader(goDevDLJson)),
		}, nil
	}))

	require.Equal(t, `matrix={"include":[{"go-version":"1.23.3"},{"go-version":"1.22.9"}]}
release-go-version=1.22.9
`, b.String())
}

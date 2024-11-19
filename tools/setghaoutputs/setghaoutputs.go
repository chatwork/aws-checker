package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// setghaoutputs is a tool to fetch the latest two stable Go versions from the Go website,
// and set the Go versions to the GitHub Actions workflow matrix and the release Go version outputs.
func main() {
	if err := setGHAOutputs(os.Stdout, http.Get); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type ghaMatrix struct {
	Include []Include `json:"include"`
}

type Include struct {
	GoVersion string `json:"go-version"`
}

func setGHAOutputs(w io.Writer, httpGet func(string) (*http.Response, error)) error {
	vers, err := getGoVersionsFromAPI(httpGet)
	if err != nil {
		return err
	}

	data, err := json.Marshal(ghaMatrix{
		Include: []Include{
			{GoVersion: vers[0]},
			{GoVersion: vers[1]},
		},
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprint(w, "matrix="); err != nil {
		return fmt.Errorf("failed to write the Go versions to the output: %w", err)
	}

	if _, err := fmt.Fprintln(w, string(data)); err != nil {
		return fmt.Errorf("failed to write the Go versions to the output: %w", err)
	}

	if _, err := fmt.Fprintf(w, "release-go-version=%s\n", vers[1]); err != nil {
		return fmt.Errorf("failed to write the release Go version to the output: %w", err)
	}

	return nil
}

type Releases []Release

type Release struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func getGoVersionsFromAPI(httpGet func(string) (*http.Response, error)) ([2]string, error) {
	url := "https://go.dev/dl/?mode=json"

	res, err := httpGet(url)
	if err != nil {
		return [2]string{}, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return [2]string{}, err
	}

	return getGoVersions(data)
}

func getGoVersions(data []byte) ([2]string, error) {
	var releases Releases
	if err := json.Unmarshal(data, &releases); err != nil {
		return [2]string{}, err
	}

	type semver struct {
		major, minor, patch int
	}

	var latestToOlder []semver
	for _, r := range releases {
		if !r.Stable {
			continue
		}

		v := strings.TrimPrefix(r.Version, "go")

		splits := strings.Split(v, ".")
		if len(splits) != 3 {
			continue
		}

		major, err := strconv.Atoi(splits[0])
		if err != nil {
			continue
		}

		minor, err := strconv.Atoi(splits[1])
		if err != nil {
			continue
		}

		patch, err := strconv.Atoi(splits[2])
		if err != nil {
			continue
		}

		latestToOlder = append(latestToOlder, semver{major, minor, patch})
	}

	if len(latestToOlder) < 2 {
		return [2]string{}, fmt.Errorf("could not find two stable Go versions")
	}

	// Sort the versions from the latest to the older.
	sort.Slice(latestToOlder, func(i, j int) bool {
		if latestToOlder[i].major != latestToOlder[j].major {
			return latestToOlder[i].major > latestToOlder[j].major
		}
		if latestToOlder[i].minor != latestToOlder[j].minor {
			return latestToOlder[i].minor > latestToOlder[j].minor
		}
		return latestToOlder[i].patch > latestToOlder[j].patch
	})

	return [2]string{
		fmt.Sprintf("%d.%d.%d", latestToOlder[0].major, latestToOlder[0].minor, latestToOlder[0].patch),
		fmt.Sprintf("%d.%d.%d", latestToOlder[1].major, latestToOlder[1].minor, latestToOlder[1].patch),
	}, nil
}

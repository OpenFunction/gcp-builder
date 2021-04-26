// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Implements go/runtime buildpack.
// The runtime buildpack installs the Go toolchain.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	// goVersionURL is a URL to a JSON file that contains the latest Go version names.
	goVersionURL   = "https://golang.org/dl/?mode=json"
	goCNVersionURL = "https://golang.google.cn/dl/?mode=json"
	goPkgName      = "go%s.linux-amd64.tar.gz"
	goLayer        = "go"
	versionKey     = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride(ctx, "go"); result != nil {
		return result, nil
	}

	if ctx.HasAtLeastOne("*.go") {
		return gcp.OptIn("found .go files"), nil
	}
	return gcp.OptOut("no .go files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return err
	}
	grl := ctx.Layer(goLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)

	// Check metadata layer to see if correct version of Go is already installed.
	metaVersion := ctx.GetMetadata(grl, versionKey)
	if version == metaVersion {
		ctx.CacheHit(goLayer)
	} else {
		ctx.CacheMiss(goLayer)
		ctx.ClearLayer(grl)

		// Install Go in layer.
		ctx.Logf("Installing Go v%s", version)
		goPkg := filepath.Join(ctx.BuildpackRoot(), fmt.Sprintf(goPkgName, version))
		command := fmt.Sprintf("tar --directory %s -xzf %s --strip-components=1", grl.Path, goPkg)
		ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
		ctx.SetMetadata(grl, versionKey, version)
	}

	return nil
}

func runtimeVersion(ctx *gcp.Context) (string, error) {
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	if version := golang.GoModVersion(ctx); version != "" {
		ctx.Logf("Using runtime version from go.mod: %s", version)
		return version, nil
	}
	version, err := latestGoVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("getting latest version: %w", err)
	}
	ctx.Logf("Using latest runtime version: %s", version)
	return version, nil
}

type goReleases []struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// latestGoVersion returns the latest version of Go
func latestGoVersion(ctx *gcp.Context) (string, error) {
	result := ctx.Exec([]string{"curl", "--connect-timeout", "3", "--fail", "--show-error", "--silent", "--location", goVersionURL}, gcp.WithUserAttribution)
	if result == nil {
		result = ctx.Exec([]string{"curl", "--connect-timeout", "3", "--fail", "--show-error", "--silent", "--location", goCNVersionURL}, gcp.WithUserAttribution)
	}
	return parseVersionJSON(result.Stdout)
}

func parseVersionJSON(jsonStr string) (string, error) {
	releases := goReleases{}
	if err := json.Unmarshal([]byte(jsonStr), &releases); err != nil {
		return "", fmt.Errorf("parsing JSON response from URL %q: %v", goVersionURL, err)
	}

	for _, release := range releases {
		if !release.Stable {
			continue
		}
		if v := strings.TrimPrefix(release.Version, "go"); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("parsing latest stable version from %q", goVersionURL)
}

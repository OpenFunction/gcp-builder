// Copyright 2021 Google LLC
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

package main

import (
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name string
		env  []string
		want int
	}{
		{
			name: "env var set",
			env:  []string{"GOOGLE_CLEAR_SOURCE=true"},
			want: 0,
		},
		{
			name: "GOOGLE_CLEAR_SOURCE not set",
			want: 100,
		},
		{
			name: "GOOGLE_CLEAR_SOURCE set and devmode enabled",
			env: []string{
				"GOOGLE_CLEAR_SOURCE=true",
				"GOOGLE_DEVMODE=true",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gcp.TestDetect(t, detectFn, tc.name, map[string]string{}, tc.env, tc.want)
		})
	}
}

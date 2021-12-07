// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statequery

// Type to hold all state per execution
type State struct {
	Reservations map[string]Reservation `json:"reservations"`
}

// Type for individual reservation data
type Reservation struct {
	Name              string   `json:"name"`
	Location          string   `json:"location"`
	Slots             float64  `json:"slots"`
	Projects          []string `json:"projects"`
	Jobs              []Job    `json:"jobs"`
	NumJobs           int      `json:"num_jobs"`
	TotalUsage        float64  `json:"total_usage"`
	TotalUsageCeiling int      `json:"total_usage_ceiling"`
	ThresholdBreached bool     `json:"threshold_breached"`
	Percentage        string   `json:"percentage"`
}

// Type for job data
type Job struct {
	Name  string  `json:"name"`
	Usage float64 `json:"usage"`
}

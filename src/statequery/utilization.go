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

import (
	"fmt"
	"math"
)

// Compute total utilization statistics per reservation and add them to the state
func (state *State) ComputeUtilization(threshold float64) {
	for id, reservation := range state.Reservations {

		// Total number of running jobs per reservation
		reservation.NumJobs = len(reservation.Jobs)

		// Total slot usage across all jobs per reservation
		for _, job := range reservation.Jobs {
			reservation.TotalUsage += job.Usage
		}

		// Utilization factor
		utilization := reservation.TotalUsage / reservation.Slots

		// Set breach flag if utilization crosses threshold
		if utilization >= threshold {
			reservation.ThresholdBreached = true
		}

		// Round slot usage up to natural ceiling
		reservation.TotalUsageCeiling = int(math.Ceil(reservation.TotalUsage))

		// Format printable utilization percentage
		reservation.Percentage = fmt.Sprintf("%.2f", utilization*100.0)
		state.Reservations[id] = reservation
	}
}

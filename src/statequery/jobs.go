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
	"context"
	"fmt"
	"log"
	"sync"

	bigquerySDK "google.golang.org/api/bigquery/v2"
)

// Retrieve job info by querying each project with an active assignment for
// jobs in 'RUNNING' state and add their stats to the state.
func (state *State) RetrieveJobs(ctx context.Context) error {
	// Create shared BQ client
	client, err := bigquerySDK.NewService(ctx)
	if err != nil {
		log.Printf("failed to initialize BigQuery client: %v\n", err)
	}

	for _, reservation := range state.Reservations {

		// Create sync & comms for concurrent invokations
		ch := make(chan Job)
		var wg sync.WaitGroup
		wg.Add(len(reservation.Projects))

		// Retrieve job stats for each assigned project
		for _, project := range reservation.Projects {
			// Create a per-project routine to avoid blocking on I/O during API calls
			go retrieveJobsProject(ctx, client, project, reservation, ch, &wg)
		}

		go func() {
			// Synchronize routines and close channel
			wg.Wait()
			close(ch)
		}()

		// Read found job stats into state
		for job := range ch {
			reservation.Jobs = append(reservation.Jobs, job)
		}
		id := fmt.Sprintf("%s.%s", reservation.Location, reservation.Name)
		state.Reservations[id] = reservation
	}
	return nil
}

// Routine to retrieve BQ job stats for a given project
func retrieveJobsProject(ctx context.Context, client *bigquerySDK.Service, project string, reservation Reservation, ch chan<- Job, wg *sync.WaitGroup) {
	// Defer completion signal on wait group
	defer wg.Done()

	// Query BQ API for jobs from all users in given project with 'RUNNING' state.
	list, err := client.Jobs.List(project).AllUsers(true).StateFilter("running").Do()
	if err != nil {
		log.Printf("failed to get BigQuery jobs: %v\n", err)
	}

	// Iterate jobs lists
	for _, job := range list.Jobs {
		// Refresh job object for current state and details jobs statistics.
		current, err := client.Jobs.Get(project, job.JobReference.JobId).Do()
		if err != nil {
			log.Printf("failed to refresh job: %v\n", err)
		}

		// Switch on job types and ignore everything but 'QUERY' jobs.
		switch current.Configuration.JobType {
		case "LOAD":
			log.Printf("skipping job of LOAD type: %v\n", current.JobReference.JobId)
			continue
		case "COPY":
			log.Printf("skipping job of COPY type: %v\n", current.JobReference.JobId)
			continue
		case "EXTRACT":
			log.Printf("skipping job of EXTRACT type: %v\n", current.JobReference.JobId)
			continue
		case "UNKNOWN":
			log.Printf("skipping job of UNKNOWN type: %v\n", current.JobReference.JobId)
			continue
		case "QUERY":
			// Get query-specific stats on this job.
			stats := current.Statistics.Query

			// Find the latest statistics sample in the job timeline
			latestSnapshot := &bigquerySDK.QueryTimelineSample{}
			for _, sample := range stats.Timeline {
				if sample.ElapsedMs > latestSnapshot.ElapsedMs {
					latestSnapshot = sample
				}
			}

			runtimeMillis := latestSnapshot.ElapsedMs
			slotMillis := latestSnapshot.TotalSlotMs

			// Safely check stats before division
			if runtimeMillis == 0 || slotMillis == 0 {
				log.Printf("failed to get job runtime stats, skipping %v\n", job.JobReference.JobId)
				continue
			}
			// Compute slot usage by eliminating time
			slots := float64(slotMillis) / float64(runtimeMillis)

			// Double check if the job's reservation matches the one we are looking for
			fullReservationId := fmt.Sprintf("%s:%s.%s", project, reservation.Location, reservation.Name)
			if current.Statistics.ReservationId != fullReservationId {
				log.Printf("warn: detected missing reservation ID or mismatch: expected %s, found %s, on job %s", fullReservationId, current.Statistics.ReservationId, job.JobReference.JobId)
			}

			// All good. Push job down the channel.
			ch <- Job{
				Name:  current.Id,
				Usage: slots,
			}
		}
	}
}

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
	"strings"
	"sync"

	reservationSDK "cloud.google.com/go/bigquery/reservation/apiv1"
	reservationPB "cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	iterator "google.golang.org/api/iterator"
)

// Retrieves BQ reservations from current (admin) project and adds them to the state
func (state *State) RetrieveReservations(ctx context.Context, project string, locations []string) error {
	state.Reservations = make(map[string]Reservation)

	// Create shared BQ reservations client
	client, err := reservationSDK.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create sync & comms for concurrent invokations
	ch := make(chan Reservation)
	var wg sync.WaitGroup
	wg.Add(len(locations))

	// Retrieve BQ reservations from every configure region/multi-region
	for _, location := range locations {
		// Create a per-region routine to avoid blocking on I/O during API calls
		go retrieveReservationLocation(ctx, client, project, location, ch, &wg)
	}

	go func() {
		// Synchronize routines and close channel
		wg.Wait()
		close(ch)
	}()

	// Read found reservations into state
	for reservation := range ch {
		id := fmt.Sprintf("%s.%s", reservation.Location, reservation.Name)
		state.Reservations[id] = reservation
	}

	return nil
}

// Routine to retrieve BQ reservations for a given region/location
func retrieveReservationLocation(ctx context.Context, client *reservationSDK.Client, project string, location string, ch chan<- Reservation, wg *sync.WaitGroup) {
	// Defer completion signal on wait group
	defer wg.Done()

	fmt.Printf("Checking -> projects/%s/locations/%s", project, location)
	// Create request on specific location
	request := &reservationPB.ListReservationsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", project, location),
	}

	// Execute API call and depaginate responses
	it := client.ListReservations(ctx, request)
	for {
		response, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("error retrieving reservations: %v\n", err)
			break
		}

		// Break up full reservation resource ID
		tokens := strings.Split(response.Name, "/")
		name := tokens[len(tokens)-1]

		if name == "default" {
			// Skip default (on-demand) reservation
			continue
		}

		log.Printf("found reservation: %s\n", response.Name)
		// Push reservation down the channel
		ch <- Reservation{
			Name:     name,
			Slots:    float64(response.SlotCapacity),
			Location: location,
		}
	}
}

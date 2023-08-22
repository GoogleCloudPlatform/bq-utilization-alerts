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

	cloudresourcemanagerSDK "google.golang.org/api/cloudresourcemanager/v3"
)

// Retrieves assignments for each reservation and adds it to the state.
// Reservations can be assigned to projects, folders and orgs.
//
// WARNING: currently, only project assignments are resolved.
func (state *State) RetrieveAssignments(ctx context.Context, project string, cache *Cache) error {
	// Create shared BQ reservations client
	resClient, err := reservationSDK.NewClient(ctx)
	if err != nil {
		return err
	}
	defer resClient.Close()

	// Create shared resource manager client
	manClient, err := cloudresourcemanagerSDK.NewService(ctx)
	if err != nil {
		return err
	}

	// Create sync for concurrent invokations
	var wg sync.WaitGroup
	wg.Add(len(state.Reservations))
	for _, reservation := range state.Reservations {
		// Create a per-reservation routine to avoid blocking on I/O during API calls
		go retrieveAssignmentReservation(ctx, resClient, manClient, cache, project, state, reservation, &wg)
	}

	// Synchronize routines
	wg.Wait()

	return nil
}

// Routine to retrieve assignments for a single reservation
func retrieveAssignmentReservation(ctx context.Context, resClient *reservationSDK.Client, manClient *cloudresourcemanagerSDK.Service, cache *Cache, project string, state *State, reservation Reservation, wg *sync.WaitGroup) {
	// Defer completion signal on wait group
	defer wg.Done()

	// Create request on specific (regional) reservation
	request := &reservationPB.ListAssignmentsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s/reservations/%s", project, reservation.Location, reservation.Name),
	}

	// Execute API call and depaginate responses
	it := resClient.ListAssignments(ctx, request)
	for {
		response, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("error retrieving assignments: %v\n", err)
			break
		}

		log.Printf("found assignment: %s\n", response.Name)

		// Break up full assignee ID
		tokens := strings.Split(response.Assignee, "/")
		assigneeType := tokens[0]
		assigneeName := tokens[1]

		// Retrieve all project children under an assignee
		children, err := retrieveResourceChildren(manClient, cache, assigneeType, assigneeName)
		if err != nil {
			log.Printf("error retrieving project children: %v\n", err)
		}

		// Add all project children to the state
		reservation.Projects = append(reservation.Projects, children...)
	}
	// Remove duplicates
	reservation.Projects = trimDuplicates(reservation.Projects)

	id := fmt.Sprintf("%s.%s", reservation.Location, reservation.Name)
	state.Reservations[id] = reservation
}

// Recursively traverses resources in Cloud Resource Manager to find all children project IDs
// given a particular parent resource.
func retrieveResourceChildren(manClient *cloudresourcemanagerSDK.Service, cache *Cache, resourceType string, resourceName string) ([]string, error) {
	// Qualified name of current parent
	parent := fmt.Sprintf("%s/%s", resourceType, resourceName)

	switch resourceType {
	case "projects":
		// Resource is a project, only return this particular resource
		return []string{
			resourceName,
		}, nil
	case "organizations":
		// Resource is an organization, resolve all children using Resource Manager
		fallthrough // Treat same as folders
	case "folders":
		// Resource is a folder, resolve all children using Resource Manager

		// Attempt cache resolution
		result, err := cache.Get(parent)
		if err == nil {
			// Cache hit, return early
			log.Printf("cache hit for key: %s\n", parent)
			return result, nil
		}

		// Cache miss
		log.Printf("cache miss: %v\n", err)

		// Resolve children of type folder
		folders, err := manClient.Folders.List().Parent(parent).Do()
		if err != nil {
			return nil, err
		}
		for _, folder := range folders.Folders {
			tokens := strings.Split(folder.Name, "/")
			folderName := tokens[1]
			children, err := retrieveResourceChildren(manClient, cache, "folders", folderName)
			if err != nil {
				return nil, err
			}
			result = append(result, children...)
		}

		// Resolve children of type project
		projects, err := manClient.Projects.List().Parent(parent).Do()
		if err != nil {
			return nil, err
		}
		for _, project := range projects.Projects {
			children, err := retrieveResourceChildren(manClient, cache, "projects", project.ProjectId)
			if err != nil {
				return nil, err
			}
			result = append(result, children...)
		}

		// Update cache
		cache.Add(parent, result)

		return result, nil
	default:
		return nil, fmt.Errorf("unexpected assignee resource type: %s", resourceType)
	}
}

// Trim duplicate strings from a slice
func trimDuplicates(s []string) []string {
	occurrenceMap := make(map[string]bool, len(s))
	var uniqueItems []string

	for _, item := range s {
		if item != "" {
			if !occurrenceMap[item] {
				occurrenceMap[item] = true
				uniqueItems = append(uniqueItems, item)
			}
		}
	}
	return uniqueItems
}

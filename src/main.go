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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"main/statequery"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Type for global configuration data
type config struct {
	port      string
	project   string
	locations []string
	threshold float64
	bucket    string
}

func main() {
	cfg := config{}
	cfg.configure()

	// Initialize empty resource manager cache with maxTTL
	// Note that the cache is unlikely to live this long when executed in some environments.
	// e.g. Cloud Run (with --min-instances 0) is likely to regularly dispose the serving
	// instances. The maxTTL setting helps with executing this service on long-lived
	// infrastructure, where it is required to eventually refreshed the cache entries.
	cache := &statequery.Cache{}
	cache.Initialize(time.Hour)

	ctx := context.Background()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// Always respond with JSON
		w.Header().Set("Content-Type", "application/json")

		log.Println("starting analysis")

		// Start from a clean slate and track state
		state := statequery.State{}

		// Retrieve all BQ reservations from current (admin) project
		err := state.RetrieveReservations(ctx, cfg.project, cfg.locations)
		if err != nil {
			log.Fatalf("failed to retrieve reservations: %v\n", err)
		}

		// Abort if no reservations have been found
		if len(state.Reservations) == 0 {
			json.NewEncoder(w).Encode(state)
			return
		}

		// Retrieve all assignments for each reservation
		err = state.RetrieveAssignments(ctx, cfg.project, cache)
		if err != nil {
			log.Fatalf("failed to retrieve assignments: %v\n", err)
		}

		// Retrieve info for all running jobs from projects with reservations
		err = state.RetrieveJobs(ctx)
		if err != nil {
			log.Fatalf("failed to retrieve jobs: %v\n", err)
		}

		// Compute utilization totals
		state.ComputeUtilization(cfg.threshold)

		err = state.DumpState(ctx, cfg.bucket)
		if err != nil {
			log.Fatalf("failed to dump state: %v\n", err)
		}

		// Abort if no reservation is breaching its threshold
		alert := false
		for _, reservation := range state.Reservations {
			if reservation.ThresholdBreached {
				alert = true
			}
		}
		if !alert {
			json.NewEncoder(w).Encode(state)
			return
		}

		// Render message from template
		message, err := state.RenderMessage()
		if err != nil {
			log.Fatalf("failed to render message: %v\n", err)
		}

		// Push rendered message to all defined chat webhooks
		err = statequery.SendMessage(cfg.hooks(), message)
		if err != nil {
			log.Fatalf("failed to send message: %v\n", err)
		}

		// Encode state to HTTP response
		json.NewEncoder(w).Encode(state)
	})

	log.Println("listening for connections")
	http.ListenAndServe(fmt.Sprintf(":%s", cfg.port), nil)
}

func (cfg *config) configure() {
	// Read PORT to listen on
	cfg.port = "8080"
	port := os.Getenv("PORT")
	if port != "" {
		cfg.port = port
	}

	// Set admin GCP project, which holds BQ reservations/assignments
	cfg.project = os.Getenv("GOOGLE_CLOUD_PROJECT")

	// Configure locations for resolution of BQ reservations
	cfg.locations = []string{
		"US",
		"EU",
		"asia-east1",
		"asia-east2",
		"asia-northeast1",
		"asia-northeast2",
		"asia-northeast3",
		"asia-south1",
		"asia-south2",
		"asia-southeast1",
		"asia-southeast2",
		"australia-southeast1",
		"australia-southeast2",
		"europe-central2",
		"europe-north1",
		"europe-west1",
		"europe-west2",
		"europe-west3",
		"europe-west4",
		"europe-west5",
		"europe-west6",
		"northamerica-northeast1",
		"northamerica-northeast2",
		"southamerica-east1",
		// "southamerica-west1", # Endpoint unavailable
		"us-central1",
		"us-east1",
		"us-east4",
		"us-west1",
		"us-west2",
		"us-west3",
		"us-west4",
	}

	// Sets alerting threshold for utilization alarms
	thres, err := strconv.ParseFloat(os.Getenv("USAGE_THRESHOLD"), 64)
	if err != nil {
		log.Println("failed to parse threshold from USAGE_THRESHOLD, defaulting to 0.8")
		thres = 0.8
	}
	cfg.threshold = thres

	//Bucket to dump state to
	cfg.bucket = os.Getenv("STATE_BUCKET")
}

// Get a webhook url for a given chat service.
// Unlike the rest of the configuration, this is kept dynamic to ensure that always
// the latest version of the secret webhook are loaded and used.
func (cfg *config) hooks() map[string]string {
	hooks := make(map[string]string)

	// Read secret webhooks from ENV vars
	hooks["slack"] = os.Getenv("SLACK_WEBHOOK_URL")
	hooks["gchat"] = os.Getenv("GCHAT_WEBHOOK_URL")

	//Override secret webhooks from secret volume mounts, if available
	for service := range hooks {
		file := fmt.Sprintf("/%s/webhook", service)
		_, err := os.Stat(file)
		if err == nil {
			secret, err := ioutil.ReadFile(file)
			if err != nil {
				log.Fatalf("failed to read secret from volume mount: %s\n", file)
			}
			hooks[service] = string(secret)
		}
	}
	return hooks
}

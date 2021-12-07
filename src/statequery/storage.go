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
	"encoding/json"
	"fmt"
	"log"
	"time"

	storageSDK "cloud.google.com/go/storage"
)

// Archive the current state into timestamped GCS object
func (state *State) DumpState(ctx context.Context, bucket string) error {
	// Abort if no bucket has been specified
	client, err := storageSDK.NewClient(ctx)
	if err != nil {
		log.Println("bucket for state dumps not configured, skipping...")
		return err
	}

	// Create object key with timestamp
	stamp := fmt.Sprintf("%v", time.Now().UTC().Unix())
	object := fmt.Sprintf("state-%s.json", stamp)

	// Create GCS writer for new object
	writer := client.Bucket(bucket).Object(object).NewWriter(ctx)
	defer writer.Close()
	writer.ContentType = "application/json"

	// Encode state to the bucket writer
	json.NewEncoder(writer).Encode(state)
	return nil
}

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
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"text/template"
)

// Renders the state into the text-template at './templates/message.template'
func (state *State) RenderMessage() (string, error) {
	template, err := template.ParseFiles("templates/message.template")
	if err != nil {
		return "", err
	}

	var writer bytes.Buffer
	err = template.Execute(&writer, state)
	if err != nil {
		return "", err
	}
	return writer.String(), nil
}

// Send message to configured webhooks
func SendMessage(hooks map[string]string, message string) error {
	for service, url := range hooks {
		// Skip if webhook URL is not configured
		if url == "" {
			log.Printf("webhook for %s not configured, skipping...\n", service)
			continue
		}
		log.Printf("publishing message to %s\n", service)

		// Serialize message into payload format accepted by Slack and Google Chat
		payload := map[string]string{"text": message}
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		// POST message to chat API.
		_, err = http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("failed to push message to %s: %v\n", service, err)
		}
	}
	return nil
}

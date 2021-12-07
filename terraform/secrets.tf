# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Keep pre-signed webhook URLs in secrets manager
resource "google_secret_manager_secret" "secret_slack_hook" {
  project  = local.project
  provider = google-beta

  secret_id = "${local.prefix}-secret-slack-hook"
  replication {
    automatic = true
  }
  depends_on = [
    google_project_service.secretmanager
  ]
}

resource "google_secret_manager_secret" "secret_gchat_hook" {
  project  = local.project
  provider = google-beta

  secret_id = "${local.prefix}-secret-gchat-hook"
  replication {
    automatic = true
  }
  depends_on = [
    google_project_service.secretmanager
  ]
}

resource "google_secret_manager_secret_version" "secret_slack_hook_data" {
  provider = google-beta

  secret      = google_secret_manager_secret.secret_slack_hook.name
  secret_data = "invalid"
  depends_on = [
    google_project_service.secretmanager
  ]
}

resource "google_secret_manager_secret_version" "secret_gchat_hook_data" {
  provider = google-beta

  secret      = google_secret_manager_secret.secret_gchat_hook.name
  secret_data = "invalid"
  depends_on = [
    google_project_service.secretmanager
  ]
}

resource "google_secret_manager_secret_iam_member" "secret_slack_hook_access" {
  project  = local.project
  provider = google-beta

  secret_id  = google_secret_manager_secret.secret_slack_hook.id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${google_service_account.service.email}"
  depends_on = [google_secret_manager_secret.secret_slack_hook]
}

resource "google_secret_manager_secret_iam_member" "secret_gchat_hook_access" {
  project  = local.project
  provider = google-beta

  secret_id  = google_secret_manager_secret.secret_gchat_hook.id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${google_service_account.service.email}"
  depends_on = [google_secret_manager_secret.secret_gchat_hook]
}

output "slack_secret" {
  value = google_secret_manager_secret.secret_slack_hook.name
}

output "gchat_secret" {
  value = google_secret_manager_secret.secret_gchat_hook.name
}

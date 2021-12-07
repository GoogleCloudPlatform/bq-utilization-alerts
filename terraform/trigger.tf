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


# Cloud Scheduler trigger for BQ job insertions in local (admin) project
resource "google_service_account" "local_trigger" {
  project      = local.project
  account_id   = "${local.prefix}-sa"
  display_name = "${local.prefix}-sa"
}

resource "google_cloud_run_service_iam_member" "local_trigger_sa_invoker" {
  location = local.region
  project  = local.project
  service  = google_cloud_run_service.service.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.local_trigger.email}"
}

resource "google_app_engine_application" "default" {
  project     = local.project
  provider    = google-beta
  location_id = replace(local.region, "/\\d$/", "")
}

resource "google_cloud_scheduler_job" "local_trigger" {
  name             = "${local.prefix}-scheduler"
  description      = "${local.prefix}-scheduler"
  project          = local.project
  schedule         = local.schedule
  time_zone        = "UTC"
  attempt_deadline = "30s"

  http_target {
    http_method = "POST"
    uri         = google_cloud_run_service.service.status[0].url

    oidc_token {
      service_account_email = google_service_account.local_trigger.email
    }
  }

  depends_on = [
    google_app_engine_application.default
  ]
}

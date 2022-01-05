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


# Archival bucket for state dumps
resource "google_storage_bucket" "service_bucket" {
  project                     = local.project
  name                        = "${local.prefix}-${local.project}-service-bucket"
  uniform_bucket_level_access = true
  location                    = "EU"
  force_destroy               = true
}


# Service Account to read BQ resources and write state dumps
resource "google_service_account" "service" {
  project      = local.project
  account_id   = "${local.prefix}-service"
  display_name = "${local.prefix}-service"
}

resource "google_project_iam_member" "service_sa_bigquery" {
  project = local.project
  role    = "roles/bigquery.resourceViewer"
  member  = "serviceAccount:${google_service_account.service.email}"
}

resource "google_storage_bucket_iam_member" "service_sa_storage" {
  bucket = google_storage_bucket.service_bucket.name
  role   = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.service.email}"
}


# Alerting service on Cloud Run
resource "google_cloud_run_service" "service" {
  project  = local.project
  provider = google-beta
  name     = "${local.prefix}-service"
  location = local.region
  template {
    spec {
      service_account_name = google_service_account.service.email
      containers {
        image = "gcr.io/${local.project}/bq-utilization-alerts"
        volume_mounts {
          name       = "slack"
          mount_path = "/slack"
        }
        volume_mounts {
          name       = "gchat"
          mount_path = "/gchat"
        }
        env {
          name  = "GOOGLE_CLOUD_PROJECT"
          value = local.project
        }
        env {
          name  = "SLOT_USAGE_THRESHOLD"
          value = "0.8"
        }
        env {
          name  = "STATE_BUCKET"
          value = google_storage_bucket.service_bucket.name
        }
        resources {
          limits = {
            memory = "256Mi"
            cpu    = "1000m"
          }
        }
      }
      volumes {
        name = "slack"
        secret {
          secret_name = google_secret_manager_secret.secret_slack_hook.secret_id
          items {
            key  = "latest"
            path = "webhook"
          }
        }
      }
      volumes {
        name = "gchat"
        secret {
          secret_name = google_secret_manager_secret.secret_gchat_hook.secret_id
          items {
            key  = "latest"
            path = "webhook"
          }
        }
      }
    }
  }
  metadata {
    annotations = {
      "run.googleapis.com/ingress"      = "all"
      "run.googleapis.com/launch-stage" = "BETA"
    }
  }
  traffic {
    percent         = 100
    latest_revision = true
  }
  depends_on = [
    google_secret_manager_secret_version.secret_slack_hook_data,
    google_secret_manager_secret_version.secret_gchat_hook_data,
    google_secret_manager_secret_iam_member.secret_slack_hook_access,
    google_secret_manager_secret_iam_member.secret_gchat_hook_access,
  ]
}

output "service_endpoint" {
  value = google_cloud_run_service.service.status[0].url
}

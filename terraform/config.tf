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


# Configuration to customize
locals {
  prefix         = "bq-utilization-alerts"
  region         = "europe-west1"
  project        = "<PROJECT_ID>"
  project_number = "<PROJECT_NUMBER>"
  schedule       = "*/5 * * * *" # Every 5 minutes
  scan_projects = toset([
  ]) # Add project IDs
  scan_organizations = toset([
  ]) # Add org numbers
}

terraform {
  backend "gcs" {
    bucket = "<PROJECT_ID>-terraform-state"
    prefix = "terraform-state"
  }
}

provider "google-beta" {
  region = local.region
}

provider "google" {
  region = local.region
}

# Enabling required services #
resource "google_project_service" "cloudrun" {
  project            = local.project
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudscheduler" {
  project            = local.project
  service            = "cloudscheduler.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "secretmanager" {
  project            = local.project
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "reservations" {
  project            = local.project
  service            = "bigqueryreservation.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "resourcemanager" {
  project            = local.project
  service            = "cloudresourcemanager.googleapis.com"
  disable_on_destroy = false
}

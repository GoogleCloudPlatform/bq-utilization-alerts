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


# Grant service SA permissions to query BigQuery in projects
resource "google_project_iam_member" "scan_bigquery" {
  for_each = local.scan_projects
  project  = each.key
  role     = "roles/bigquery.resourceViewer"
  member   = "serviceAccount:${google_service_account.service.email}"
}

# (Alternatively) Grant service SA permissions to query BigQuery and folders/projects in org
resource "google_organization_iam_member" "scan_bigquery" {
  for_each = local.scan_organizations
  org_id   = each.key
  role     = "roles/bigquery.resourceViewer"
  member   = "serviceAccount:${google_service_account.service.email}"
}

resource "google_organization_iam_member" "scan_folders" {
  for_each = local.scan_organizations
  org_id   = each.key
  role     = "roles/resourcemanager.folderViewer"
  member   = "serviceAccount:${google_service_account.service.email}"
}

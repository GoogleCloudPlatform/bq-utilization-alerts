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

init:
	gsutil mb gs://$$(gcloud config get-value project)-terraform-state/ || exit 0
	terraform -chdir=terraform init

local:
	cd src; GOOGLE_CLOUD_PROJECT=$$(gcloud config get-value project) go run main.go

deploy:
	terraform -chdir=terraform apply

destroy:
	terraform -chdir=terraform destroy

build:
	gcloud services enable cloudbuild.googleapis.com
	gcloud builds submit --tag gcr.io/$$(gcloud config get-value project)/bq-utilization-alerts ./src

all: init build deploy

.PHONY: init local deploy destroy build

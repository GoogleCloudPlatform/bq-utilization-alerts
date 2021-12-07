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

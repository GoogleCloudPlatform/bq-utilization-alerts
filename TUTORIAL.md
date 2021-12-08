<!-- 
Copyright 2021 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# BigQuery Utilization Alerts

## Introduction

This tutorial walks you through the creation of the required infrastructure and the BigQuery Utilization Alerts service.

For more information on the general solution architecture and the required tools, please read the README.

<walkthrough-editor-open-file filePath="README.md">
Check the README.md
</walkthrough-editor-open-file>

## Reference Pattern for BigQuery reservations

Google recommends creating a [dedicated project](https://cloud.google.com/bigquery/docs/reservations-workload-management#admin-project) for managing BigQuery slot commitments, reservations and assignments. This will make it easier for you to manage your reservations in a centralized place and also helps with controlling IAM-based access to these resources.

You will be setting up such an admin project over the course of this tutorial, including all of the BigQuery reservations resources to showcase how the service interacts with these resources. Additionally, you'll be setting up two more projects, which will be assigned to use the reserved BigQuery slot capacity.

**Disclaimer**: This tutorial will instruct you to also create BigQuery flex slots commitments in the admin project. These flex slots can incur a considerable amount of cost, so make sure that you decommision them as soon as you no longer need them.

## Creating the BigQuery Commitments, Reservations and Assignments

### Setting up the Admin Project

Let's begin by creating a new (admin) project for this tutorial.

<walkthrough-project-setup billing></walkthrough-project-setup>

Configure the gcloud-cli to use the same (admin) project:

```terminal
gcloud config set project <PROJECT_ID>
```

Once the admin project has been selected, you can create reservations in it. We will purchase dedicated slots (called commitments), create pools of slots (called reservations), and assign BigQuery compute projects (resources) to those reservations.

You might use the different projects to represent different workloads or business units within your organization.

Next, create **two additional projects** and make note of their Project-IDs. You will need these later on. These projects will be home to the workloads or business units using BigQuery.

### Enable the BigQuery Reservation API

The BigQuery Reservation API needs to be enabled.

<walkthrough-enable-apis apis="bigqueryreservation.googleapis.com"></walkthrough-enable-apis>

Alternatively, you can enable the Reservations API directly from the CLI:

```terminal
gcloud services enable bigqueryreservation.googleapis.com
```

### Set Variables

First, you will need to assign a couple of variables that will be used during this part of the tutorial.

Set to the current (admin) project:

```terminal
export ADMIN_PROJECT_ID="$(gcloud config get-value project)"
```

Replace these with the Project-IDs for the previously created addtional projects:

```terminal
export PROJECT_A_SLOTS=<PROVIDE-WORKLOAD-PROJECT-A-ID>
```

```terminal
export PROJECT_B_SLOTS=<PROVIDE-WORKLOAD-PROJECT-B-ID>
```

Commitments and reservations will be created in the US multi-region:

```terminal
export LOCATION=US
export RESERVATION_NAME_1=r1
export RESERVATION_NAME_2=r2
```

### Purchase a Flex Slots Commitment

Now it's time to purchase Flex Slots. You can learn more about them[[here]](https://cloud.google.com/bigquery/docs/reservations-details).

Purchase 100 Flex Slots on the current (admin) project:

```terminal
bq mk \
  --location=$LOCATION \
  --capacity_commitment \
  --plan=FLEX \
  --slots=100
```

### Create a Reservation on the Commitment

Next, you'll split up the commitment into two reservations of equal size:

```terminal
bq mk \
  --location=$LOCATION \
  --reservation \
  --slots=50 \
  --ignore_idle_slots=false \
  $RESERVATION_NAME_1
```

```terminal
bq mk \
  --location=$LOCATION \
  --reservation \
  --slots=50 \
  --ignore_idle_slots=false \
  $RESERVATION_NAME_2
```

### Assign the Reservations to the Workload Projects

You can now assign a reservation to the workload projects.

Let's map the first reservation to the first workload project...

```terminal
bq mk \
  --location=$LOCATION \
  --reservation_assignment \
  --reservation_id=$RESERVATION_NAME_1 \
  --job_type=QUERY \
  --assignee_id=$PROJECT_A_SLOTS \
  --assignee_type=PROJECT
```

... and the other to the second workload project:

```terminal
bq mk \
  --location=$LOCATION \
  --reservation_assignment \
  --reservation_id=$RESERVATION_NAME_2 \
  --job_type=QUERY \
  --assignee_id=$PROJECT_B_SLOTS \
  --assignee_type=PROJECT
```

At this point the two workload projects are setup to consume slot capacity from the provisioned commitment. If you'd like, you can use the Google Cloud Console and run some BigQuery jobs from one of the workload projects.

<walkthrough-menu-navigation sectionId="BIGQUERY_SECTION"></walkthrough-menu-navigation>

## Create Slack/GChat webhooks

The service you are about to deploy acts like a bot and will occasionally post messages to a Chat service like Google Chat or Slack. In order to do this, you will need to setup webhooks in your preferred Chat application. Please refer to the external documentation to do this.

[Instructions for setting up webhooks in Slack](https://api.slack.com/messaging/webhooks)

[Instructions for setting up webhooks in Google Chat](https://developers.google.com/chat/how-tos/webhooks)

Copy the created pre-signed webhook URLs as you will need them later on.

## Local Service Development

The utilization analysis service is written in Go and users are able to test and run their code locally.

<walkthrough-editor-open-file filePath="src/main.go">
Have a look at the code in src/main.go
</walkthrough-editor-open-file>

Export the current project to the environment:

```terminal
export GOOGLE_CLOUD_PROJECT="$(gcloud config get-value project)"
```

The service can be configured, using environment variables. The `SLOT_USAGE_THRESHOLD` sets the minimum slot utilization for the service to actually send notifications on Slack or Google Chat:

```terminal
export SLOT_USAGE_THRESHOLD="0.8"
```

(Optional) If you'd like to have the service dump its analysis to blob storage, you can specify a GCS bucket:

```terminal
export STATE_BUCKET=<PROVIDE-BUCKET-NAME>
```

Next, let's replace and export the Slack or Google Chat webhook URLs (or both) to the environment:

```terminal
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/xxxxxxxxxx/xxxxxxxxxx/xxxxxxxxxxxxxxxxxxxxx"
```

```terminal
export GCHAT_WEBHOOK_URL="https://chat.googleapis.com/v1/spaces/xxxxxxxxxxx/messages?key=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx&token=xxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

Finally, we can execute the service locally:

```terminal
cd src
go run main.go
cd ..
```

To test the service locally, you need to invoke its endpoint with a HTTP POST request. You can do this with `curl`:

```terminal
curl -X POST \
  http://localhost:8080
```

## Build the Container Image

Enable Cloud Build to build the service container image remotely.

<walkthrough-enable-apis apis="cloudbuild.googleapis.com"></walkthrough-enable-apis>

Alternatively, you can enable the Cloud Build API directly:

```terminal
gcloud services enable cloudbuild.googleapis.com
```

Upload the build context in `./src` and run the container integration on Cloud Build:

```terminal
gcloud builds submit --tag gcr.io/$(gcloud config get-value project)/bq-utilization-alerts ./src
```

## Deploying the Infrastructure to the Admin Project

Now that the container image for the Cloud Run service has been built, it's time to deploy the central infrastructure and the service to the admin project.

Hashicorp's `terraform` is used to setup infrastructure as code. To track the terraform state remotely, we first need to create a GCS bucket:

```terminal
gsutil mb gs://$(gcloud config get-value project)-terraform-state/
```

Update `terraform/config.tf` with the name of the newly created terraform state bucket.

<walkthrough-editor-select-line filePath="terraform/config.tf"
                                startCharacterOffset="0"
                                endCharacterOffset="0"
                                startLine="31"
                                endLine="32">
Edit `terraform/config.tf`
</walkthrough-editor-select-line>

Update `terraform/config.tf` to match the configuration of your environment.

<walkthrough-editor-select-line filePath="terraform/config.tf"
                                startCharacterOffset="0"
                                endCharacterOffset="0"
                                startLine="20"
                                endLine="22">
Edit `terraform/config.tf`
</walkthrough-editor-select-line>

After completing configuration, `terraform` needs to initialized with the Google Cloud providers:

```terminal
terraform -chdir=terraform init
```

Finally, you can plan the terraform deployment...

```terminal
terraform -chdir=terraform plan
```

... and apply it:

```terminal
terraform -chdir=terraform apply
```

Insert your presigned Slack and/or Google Chat webhook URLs into new versions of the created secrets in Google Cloud Secrets Manager:

```terminal
echo -n  "https://chat.googleapis.com/v1/spaces/....." | gcloud secrets versions add "$(terraform -chdir=terraform output -raw gchat_secret)" --data-file=-
```

```terminal
echo -n  "https://hooks.slack.com/services/....." | gcloud secrets versions add "$(terraform -chdir=terraform output -raw slack_secret)" --data-file=-
```

## Checking the Triggers

The infrastructure has been setup and the scheduler should already periodically invoke the service in the central (admin) project.

*Note*: Per default, the scheduler is set to trigger the service every 5 minutes. You might have to wait a little to see the first invokations.

Check the execution logs of the Cloud Run service.

<walkthrough-menu-navigation sectionId="CLOUD_RUN_SECTION"></walkthrough-menu-navigation>

## Add Remote BigQuery Workload Projects

Finally, in order to be able to query for BigQuery jobs in the workload projects, the service account of the service needs the necessary permissions to query the projects.

To do this, you need to add configuration to the Terraform manifest. Update `terraform/config.tf` and add the Project-IDs of the two previously created workload projects to the set of `local.scan_projects`:

<walkthrough-editor-select-line filePath="terraform/config.tf"
                                startCharacterOffset="0"
                                endCharacterOffset="0"
                                startLine="23"
                                endLine="25">
Edit `terraform/config.tf`
</walkthrough-editor-select-line>

Alternatively, you can add the unique number of your organization and give the service access to all projects in your organization by modifying the set `local.scan_organizations`:

<walkthrough-editor-select-line filePath="terraform/config.tf"
                                startCharacterOffset="0"
                                endCharacterOffset="0"
                                startLine="25"
                                endLine="27">
Edit `terraform/config.tf`
</walkthrough-editor-select-line>

Afterwards, you can re-apply the changed configuration with Terraform:

```terminal
terraform -chdir=terraform apply
```

## It's done! 🎉

The service in the admin project should now also be able to query for BigQuery jobs in the configured workload projects.

## Cleaning Up

To clean up all the resources in reverse sequence, follow along.

### Destroying the Infrastructure

Use Terraform to destroy all created resources:

```terminal
terraform -chdir=terraform destroy
```

### Cleaning up BigQuery Assignment, Reservations and Commitments

Next, you have to clean up the created BigQuery resources.

Remove assignments from the workload projects:

```terminal
export ASSIGNMENT_ID_1=$(bq ls --location=$LOCATION --reservation_assignment $ADMIN_PROJECT_ID:US.r1 | awk '{if(NR>2)print}' | awk '{print $1}')
```

```terminal
export ASSIGNMENT_ID_2=$(bq ls --location=$LOCATION --reservation_assignment $ADMIN_PROJECT_ID:US.r2 | awk '{if(NR>2)print}' | awk '{print $1}')
```

```terminal
bq rm \
--location=$LOCATION \
--reservation_assignment $ASSIGNMENT_ID_1
```

```terminal
bq rm \
--location=$LOCATION \
--reservation_assignment $ASSIGNMENT_ID_2
```

Delete reservations in the admin project:

```terminal
bq rm \
--location=$LOCATION \
--reservation $RESERVATION_NAME_1
```

```terminal
bq rm \
--location=$LOCATION \
--reservation $RESERVATION_NAME_2
```

Delete the Flex Slots commitment:

```terminal
export LATEST_COMMITMENT_ID=$(bq ls --location=$LOCATION --capacity_commitment | awk '{if(NR>2)print}' | awk '{print $1}')
```

```terminal
bq rm \
--location=$LOCATION \
--capacity_commitment $LATEST_COMMITMENT_ID
```

After deleting the Flex Slots commitment, you will no longer be charged for the costs they incur.

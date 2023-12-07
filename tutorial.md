<walkthrough-metadata>
  <meta name="title" content="Edit Jumpstart Solution and deploy tutorial " />
   <meta name="description" content="Make it mine neos tutorial" />
  <meta name="component_id" content="1361081" />
  <meta name="unlisted" content="true" />
  <meta name="short_id" content="true" />
</walkthrough-metadata>

# Customize Three-tier web app Solution

This tutorial provides the steps for you to build your own proof of concept solution based on the deployed [Three-tier web app](https://console.cloud.google.com/products/solutions/details/three-tier-web-app) Jump Start Solution (JSS) and deploy it. You can customize the Jump Start Solution (JSS) deployment by creating your own copy of the source code. You can modify the infrastructure and application code as needed and redeploy the solution with the changes.

The solution should be edited and deployed by one user at a time to avoid conflicts. Multiple users editing and updating the same deployment in the same GCP project can lead to conflicts.

## Know your solution

Here are the details of the Three-tier web app Jump Start Solution chosen by you.

Solution Guide: [here](https://cloud.google.com/architecture/application-development/three-tier-web-app)

The code for the solution is avaiable at the following location
* Infrastructure code is present as part of <walkthrough-editor-open-file filePath="./main.tf">main.tf</walkthrough-editor-open-file>
* Application code directory is located under `./src`


## Explore or Edit the solution as per your requirement

The application source code for the frontend service is present under `src/frontend` directory and for the middleware service under `src/middleware` directory.

Both these services are built as container images and deployed using cloud run. The terraform code is present in the `*.tf` files in the current directory.

As an example, you can edit the `createHandler` function in <walkthrough-editor-select-line filePath="./src/middleware/main.go" startLine="170" endLine="171" startCharacterOffset="0" endCharacterOffset="0">./src/middleware/main.go</walkthrough-editor-select-line> to add a prefix string to every TODO item by replacing `t.Title = r.FormValue("title")` with `t.Title = "Prefix " + r.FormValue("title")`.

NOTE: The changes in infrastructure may lead to reduction or increase in the incurred cost. For example, storing the container images for the services incurs [storage cost](https://cloud.google.com/container-registry/pricing).

Please note: to open your recently used workspace:
* Go to the `File` menu.
* Select `Open Recent Workspace`.
* Choose the desired workspace.


---
**Automated deployment**

Execute the <walkthrough-editor-open-file filePath="./deploy.sh">deploy.sh</walkthrough-editor-open-file> script if you want an automated deployment to happen without following the full tutorial.
This step is optional and you can continue with the full tutorial if you want to understand the individual steps involved in the script.

```bash
./deploy.sh
```

## Gather the required information for intializing gcloud command

In this step you will gather the information required for the deployment of the solution.

---
**Project ID**

Use the following command to see the projectId:

```bash
gcloud config get project
```

```
Use above output to set the <var>PROJECT_ID</var>
```

---
**Deployment Region**

```
Provide the region (e.g. us-central1) where the top level deployment resources were created for the deployment <var>REGION</var>
```

---
**Deployment Name**

Run the following command to get the existing deployment name:
```bash
gcloud infra-manager deployments list --location <var>REGION</var> --filter="labels.goog-solutions-console-deployment-name:* AND labels.goog-solutions-console-solution-id:three-tier-web-app"
```

```
Use the NAME value of the above output to set the <var>DEPLOYMENT_NAME</var>
```


## Deploy the solution


---
**Fetch Deployment details**
```bash
gcloud infra-manager deployments describe <var>DEPLOYMENT_NAME</var> --location <var>REGION</var>
```
From the output of this command, note down the input values provided in the existing deployment in the `terraformBlueprint.inputValues` section.

Also note the serviceAccount from the output of this command. The value of this field is of the form
```
projects/<var>PROJECT_ID</var>/serviceAccounts/<service-account>@<var>PROJECT_ID</var>.iam.gserviceaccount.com
```

```
Note <service-account> part and set the <var>SERVICE_ACCOUNT</var> value.
You can also set it to any exising service account.
```

---
**Assign the required roles to the service account**
```bash
while IFS= read -r role || [[ -n "$role" ]]
do \
gcloud projects add-iam-policy-binding <var>PROJECT_ID</var> \
  --member="serviceAccount:<var>SERVICE_ACCOUNT</var>@<var>PROJECT_ID</var>.iam.gserviceaccount.com" \
  --role="$role"
done < "roles.txt"
```

----
**Create container images**

NOTE: Modify the Image tags incrementally. Sample value=`1.0.0`

Execute the following command to build and push the container image for middleware and frontend:
```bash
cd ./src/middleware
gcloud builds submit --config=./cloudbuild.yaml --substitutions=_IMAGE_TAG="<var>IMAGE_TAG</var>"
cd -
cd ./src/frontend
gcloud builds submit --config=./cloudbuild.yaml --substitutions=_IMAGE_TAG="<var>IMAGE_TAG</var>"
cd -
```

Modify the `api_image` and `fe_image` value in <walkthrough-editor-select-line filePath="./main.tf" startLine="20" endLine="24" startCharacterOffset="0" endCharacterOffset="0">main.tf</walkthrough-editor-select-line> with the updated image tag.
```
locals {
  api_image = "gcr.io/<var>PROJECT_ID</var>/three-tier-app-be:<var>IMAGE_TAG</var>"
  fe_image  = "gcr.io/<var>PROJECT_ID</var>/three-tier-app-fe:<var>IMAGE_TAG</var>"
}
```

---
**Create Terraform input file**

Create an `input.tfvars` file in the current directory.

Find the sample content below and modify it by providing the respective details.
```
region="us-central1"
zone="us-central1-a"
project_id = "<var>PROJECT_ID</var>"
deployment_name = "<var>DEPLOYMENT_NAME</var>"
labels = {
  "goog-solutions-console-deployment-name" = "<var>DEPLOYMENT_NAME</var>",
  "goog-solutions-console-solution-id" = "three-tier-web-app"
}
```

---
**Deploy the solution**

Execute the following command to trigger the re-deployment.
```bash
gcloud infra-manager deployments apply projects/<var>PROJECT_ID</var>/locations/<var>REGION</var>/deployments/<var>DEPLOYMENT_NAME</var> --service-account projects/<var>PROJECT_ID</var>/serviceAccounts/<var>SERVICE_ACCOUNT</var>@<var>PROJECT_ID</var>.iam.gserviceaccount.com --local-source="."     --inputs-file=./input.tfvars --labels="modification-reason=make-it-mine,goog-solutions-console-deployment-name=<var>DEPLOYMENT_NAME</var>,goog-solutions-console-solution-id=three-tier-web-app,goog-config-partner=sc"
```

---
**Monitor the Deployment**

Execute the following command to get the deployment details.

```bash
gcloud infra-manager deployments describe <var>DEPLOYMENT_NAME</var> --location <var>REGION</var>
```

Monitor your deployment at [JSS deployment page](https://console.cloud.google.com/products/solutions/deployments?pageState=(%22deployments%22:(%22f%22:%22%255B%257B_22k_22_3A_22Labels_22_2C_22t_22_3A13_2C_22v_22_3A_22_5C_22modification-reason%2520_3A%2520make-it-mine_5C_22_22_2C_22s_22_3Atrue_2C_22i_22_3A_22deployment.labels_22%257D%255D%22))).

## Save your edits to the solution

Use any of the following methods to save your edits to the solution

---
**Download your solution in tar file**
* Click on the `File` menu
* Select `Download Workspace` to download the whole workspace on your local in compressed format.

---
**Save your solution to your git repository**

Set the remote url to your own repository
```bash 
git remote set-url origin [your-own-repo-url]
```

Review the modified files, commit and push to your remote repository branch.
<walkthrough-inline-feedback></walkthrough-inline-feedback>

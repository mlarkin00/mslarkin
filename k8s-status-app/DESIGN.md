# Goal

Generate a design guide for a web app that collects metrics and data about GKE clusters, and exposes them via a web app.

# Important Instructions ub

* Do not write any code, just develop the design as a markdown formatted document.
* ..

# Design

## Backend

* Written in Golang
* Deployed to GKE
* Uses both GKE OneMCP ([https://container.googleapis.com/mcp](https://container.googleapis.com/mcp)) as well as the GKE OSS MCP ([https://mcp.ai.mslarkin.com](https://mcp.ai.mslarkin.com)) as needed (Prefer these over command line tools like "kubectl" or "gcloud")
* Accesses relevant configuration, status, and metrics from GKE, for use by the Frontend.

## Frontend

* Written in Golang
* Uses A2UI ([https://a2ui.org](https://a2ui.org)) to receive data from the backend
* Presents a landing page where the user can input a project or list of projects
* Shows projects  as tabs
* Shows clusters as cards on the relevant tab
* Cluster Cards show
  * List of workloads by namespace (e.g. agent-ns) and type (e.g. deployment, service, etc.).  Each of these should be a separate list
* Each workload row should show:
  * Workload name
  * Status
  * other information aligning with "kubectl get X" where "X" is the resource type (deployments, services, etc.).  For example: "kubectl get deployments" returns: NAME,  READY,  UP-TO-DATE,  AVAILABLE , AGE.  So this data should be displayed for each deployment.
* Each workload row should have a "Describe" button, that presents the output of "kubectl describe \[RESOURCE\] \[NAME\]".  For example "kubectl describe deployment go-flexible-workload
* Each workload row should have a "Pods" button, that presents the list of pods for that workload, including status

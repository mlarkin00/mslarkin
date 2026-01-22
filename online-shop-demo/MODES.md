# Goal
Our goal is to highlight testing and diagnostic tools, showing how they can be used to identify and resolve issues in a kubernetes environment.  Using the online shop demo (see APP.md) as the example application we're working on, we want to be able to introduce a variety of issues (failure modes) and then use the tools to identify and resolve them.

# Modes
* Failed deployment: crashlooping container
* Failed deployment: image pull error
* Autoscaling: App did not scale up under load
* Resources: Out of memory errors
* Optimization: App utilizing a fraction of the requested resources

# Scenarios

## 1. Failed deployment: crashlooping container
* **Description:** A specific service in the Online Shop (e.g., `emailservice`) fails to start successfully and repeatedly crashes.
* **Suggested options for introduction:**
    * Edit the deployment manifest to set an invalid command or argument (e.g., `command: ["/bin/false"]`).
    * Set a required environment variable to an invalid type/value that causes a fatal exception on startup.
* **Expectation of manifestation:**
    * `kubectl get pods` shows status `CrashLoopBackOff`.
    * `kubectl logs` shows the error/exit code.

## 2. Failed deployment: image pull error
* **Description:** The cluster attempts to start a pod but cannot retrieve the container image.
* **Suggested options for introduction:**
    * Update a deployment (e.g., `currencyservice`) to use a non-existent image tag (e.g., `:v99.99-broken`).
* **Expectation of manifestation:**
    * `kubectl get pods` shows `ImagePullBackOff` or `ErrImagePull`.
    * Pod events (`kubectl describe pod`) show "Failed to pull image".

## 3. Autoscaling: App did not scale up under load
* **Description:** Load is generated against the application, but the Horizontal Pod Autoscaler (HPA) fails to provision new replicas.
* **Suggested options for introduction:**
    * **Option A:** Apply a `ResourceQuota` to the namespace that strictly limits the CPU/Memory, preventing new pods from being scheduled even if HPA requests them.
    * **Option B:** Configure HPA `maxReplicas` to a low number (e.g., 1) matching the current replica count.
* **Expectation of manifestation:**
    * **Option A:** Pods are created but stick in `Pending` state with "Forbidden" or "Exceeded quota" events.
    * **Option B:** HPA status indicates it wants to scale but is capped by `maxReplicas`.

## 4. Resources: Out of memory errors
* **Description:** A service exhausts its allocated memory limit and is terminated by the kernel.
* **Suggested options for introduction:**
    * Drastically reduce the memory limit in a Deployment's resources section (e.g., set `limits.memory: "10Mi"` for `adservice` which is written in Java and needs more).
* **Expectation of manifestation:**
    * Pod crashes with status `OOMKilled`.
    * `kubectl describe pod` shows "Reason: OOMKilled".

## 5. Optimization: App utilizing a fraction of the requested resources
* **Description:** Services are over-provisioned, reserving resources they don't need, preventing other workloads from scheduling (bin-packing inefficiency).
* **Suggested options for introduction:**
    * Increase `requests` for a lightweight service (e.g., `paymentservice`) to a high value (e.g., `cpu: "2000m"`, `memory: "4Gi"`) while the actual usage remains low.
* **Expectation of manifestation:**
    * `kubectl top pods` shows very low usage (e.g., 10m CPU).
    * `kubectl describe node` shows high "Requests" percentage, potentially leading to scale-up of nodes despite low actual cluster load.

## 6. Scale: Increased downstream latency
* **Description:** Inbound traffic increases significantly, causing high latency because a downstream service is bottlenecked by being limited to a single replica.
* **Suggested options for introduction:**
    * Increase the load generator's traffic (e.g., `USERS` env var set to `100` or more).
    * Explicitly configure a critical downstream service (e.g., `productcatalogservice`) to `replicas: 1` (even if it's the default, enforcing it ensures the bottleneck).
* **Expectation of manifestation:**
    * High latency observed in the frontend and the specific downstream service.
    * Potential timeouts or errors if the load is high enough.


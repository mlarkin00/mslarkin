# Online Shop Demo Failure Scenarios

This directory contains failure mode scenarios for the Online Shop Demo. Each scenario is designed to manifest specific issues in the Kubernetes environment for testing and training purposes.

## System Instructions

**Prerequisites:**
- `kubectl` must be configured to point to the target cluster.
- All commands should be run from the root of this directory (`online-shop-demo`).

### Baseline State
The baseline application definition is located at `baseline/release/kubernetes-manifests.yaml`.
To reset the cluster to the baseline state (except for unique artifacts like ResourceQuotas), you can apply this manifest.

## Scenarios

### 1. Failed deployment: crashlooping container
- **Description:** `emailservice` crashes on startup due to an invalid command.
- **Manifestation:** Pod status `CrashLoopBackOff`.
- **Apply:**
  ```bash
  ./failure-modes/crashloop/apply.sh
  ```
- **Revert:**
  ```bash
  kubectl apply -f baseline/release/kubernetes-manifests.yaml
  ```

### 2. Failed deployment: image pull error
- **Description:** `currencyservice` fails to pull image due to a non-existent tag.
- **Manifestation:** Pod status `ImagePullBackOff`.
- **Apply:**
  ```bash
  ./failure-modes/image-pull/apply.sh
  ```
- **Revert:**
  ```bash
  kubectl apply -f baseline/release/kubernetes-manifests.yaml
  ```

### 3. Autoscaling: App did not scale up under load
- **Description:** A `ResourceQuota` prevents the namespace from scaling up pods, mimicking an autoscaling failure or stuck deployment.
- **Manifestation:** New pods stuck in `Pending` state.
- **Apply:**
  ```bash
  ./failure-modes/autoscaling/apply.sh
  ```
- **Revert:**
  ```bash
  kubectl delete -f failure-modes/autoscaling/quota-limit.yaml
  ```

### 4. Resources: Out of memory errors
- **Description:** `adservice` has its memory limit severely restricted (`10Mi`).
- **Manifestation:** Pod status `OOMKilled`.
- **Apply:**
  ```bash
  ./failure-modes/oom/apply.sh
  ```
- **Revert:**
  ```bash
  kubectl apply -f baseline/release/kubernetes-manifests.yaml
  ```

### 5. Optimization: App utilizing a fraction of the requested resources
- **Description:** `paymentservice` requests excessive resources (`2000m` CPU, `4Gi` Memory), causing bin-packing inefficiency.
- **Manifestation:** High reserved resources vs. low usage; potential node scale-up.
- **Apply:**
  ```bash
  ./failure-modes/overprovisioning/apply.sh
  ```
- **Revert:**
  ```bash
  kubectl apply -f baseline/release/kubernetes-manifests.yaml
  ```

### 6. Scale: Increased downstream latency
- **Description:** Increased load (1000 users) combined with a bottleneck (1 replica) in `productcatalogservice` causes high latency.
- **Manifestation:** High latency in frontend and productcatalogservice.
- **Apply:**
  ```bash
  ./failure-modes/latency/apply.sh
  ```
- **Revert:**
  ```bash
  kubectl apply -f baseline/release/kubernetes-manifests.yaml
  ```

## Global Reset
To ensure the environment is completely reset to the baseline state:

```bash
# 1. Remove any added quotas
kubectl delete -f failure-modes/autoscaling/quota-limit.yaml --ignore-not-found

# 2. Re-apply baseline manifests
kubectl apply -f baseline/release/kubernetes-manifests.yaml
```

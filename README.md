# helm-controller

A simple controller built with the [Operator SDK](https://github.com/operator-framework/operator-sdk) that watches a CRD per Helm Chart within a namespace and manages installation, upgrade and deletes.

This Operator watches for CRD updates and triggers a Kubernetes job if a change is detected. The Kubernetes job executes a Docker image that runs `helm upgrade --install` with various options to make it idempotent.

To upgrade a chart you can use `kubectl apply` to modify the version in the chart CRD in your pipeline. To debug what happened you can use `kubectl logs` on the kubernetes job.

To completely remove the chart and do the equivalent of `helm delete --purge` simply delete the CRD.

The image used in the kubernetes job can be customised so you can easily add additional logic or helm plugins. This also means that Helm 3.0 will be supported on the day it goes GA.

# Installation

The default manifests create a service account, role, rolebinding and deployment that runs the operator. It is recommended to run the controller in its own namespace alongside the CRD's that it watches.

The controller will need RBAC permissions to install whatever Helm Charts it manages into whatever target namespaces that are defined.

```
kubectl create ns helm-controller
cd deploy
kubectl apply -n helm-controller -f .
```

Then to install a chart you can apply the following manifest.

```
cat <<EOF | kubectl apply -n helm-controller -f -
apiVersion: helm.kubedex.com/v1
kind: HelmChart
metadata:
  name: kubernetes-dashboard
  namespace: helm-controller
spec:
  chart: kubernetes-dashboard
  version: 1.8.0
  repo: stable
  targetNamespace: dashboard
  valuesContent: |-
    rbac.clusterAdminRole: true
    enableInsecureLogin: true
    enableSkipLogin: value: true
EOF
```

In this example we're installing the kubernetes-dashboard chart into the dashboard namespace and setting some truly dangerous values under valuesContent.


# Installation Lifecycle

The helm-controller manifests should be deployed to the Kubernetes cluster using kubectl.

* A single CRD per Helm Chart is applied via kubectl into the helm-controller namespace.
* The helm-controller watches for CRD changes and triggers a Kubernetes job per chart
* Each job executes the upgrade logic for the Helm Chart
* If a CRD is deleted for a chart the helm-controller will totally remove all resources associated with it. Including Helm Chart, old jobs and pods used for previous installations.

## Helm Chart CRD's

CRD's are the single point of truth for what Helm Charts a cluster should be running. You can view all CRD's by executing the following command:

```
kubectl get helmcharts.helm.kubedex --all-namespaces
```

Or, to look at the settings of a single CRD you can use this command:

```
kubectl get -n dashboard helmchart.helm.kubedex kubernetes-dashboard -o yaml
```

For testing and playing around purposes you can edit the CRD's directly to bump chart version or change values. On change the helm-controller will execute a Kubernetes job to apply the Helm Chart upgrade.

# Troubleshooting

* Check that the helm-controller is running on the cluster
* Get the contents of the chart CRD on the cluster using kubectl
* Check the helm-controller and CRD install job logs on the cluster using kubectl logs

To fully reset a chart you can delete the CRD. Then wait for all resources to be removed. Then apply the CRD again.

To remove all charts from a cluster you can run:

```
kubectl delete helmcharts.helm.kubedex -n helm-controller --all
```

# Credits

Heavily inspired by the [Rancher Helm Controller](https://github.com/rancher/helm-controller).

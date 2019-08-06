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

# Credits

Heavily inspired by the [Rancher Helm Controller](https://github.com/rancher/helm-controller).

# helm-controller

A simple controller built with the [Operator SDK](https://github.com/operator-framework/operator-sdk) that watches for chart CRD's within a namespace and manages installation, upgrades and deletes using Kubernetes jobs.

The helm-controller creates a Kubernetes [job image](https://github.com/Kubedex/helm-controller-jobimage) per CRD that runs `helm upgrade --install` with various options to make it idempotent. Within each job pod Helm is run in 'tillerless' mode.

To upgrade a chart you can use `kubectl` to modify the version or values in the chart CRD. To debug what happened you can use `kubectl logs` on the kubernetes job.

To completely remove the chart and do the equivalent of `helm delete --purge` simply delete the chart CRD.

The image used in the kubernetes job can be customised so you can easily add additional logic or helm plugins. You can set this by changing the `JOB_IMAGE` environment variable in the deployment manifest.

This means that Helm 3.0 will be supported on the day it goes GA.

# Installation

The default manifests create a service account, role, rolebinding and deployment that runs the operator. It is recommended to run the controller in its own namespace alongside the CRD's that it watches.

Some example manifests are available in the `deploy/` directory. 

```
kubectl apply -f deploy/
```

This will install the helm-controller into the default namespace.

Or, install using helm.

```
helm repo add kubedex https://kubedex.github.io/charts
helm repo update
helm install kubedex/helm-controller
```

Then to install a chart you can apply the following manifest.

```
cat <<EOF | kubectl apply -f -
apiVersion: helm.kubedex.com/v1
kind: HelmChart
metadata:
  name: kubernetes-dashboard
  namespace: default
spec:
  chart: stable/kubernetes-dashboard
  version: 1.8.0
  targetNamespace: kube-system
  valuesContent: |-
    rbac.clusterAdminRole: true
    enableInsecureLogin: true
    enableSkipLogin: true
EOF
```

In this example we're installing the kubernetes-dashboard chart into the kube-system namespace and setting some truly dangerous values under valuesContent.

## Private Chart Repos

In the example above we're using the pre-configured stable repo available in the default job image.

You can also specify your own private chart repo as follows.

```
apiVersion: helm.kubedex.com/v1
kind: HelmChart
metadata:
  name: myprivatechartname
  namespace: default
spec:
  chart: myprivatechartname
  repo: https://user:password@charts.example.com
  version: 1.0.0
  targetNamespace: default
```

The default job image works with https registries and basic auth. You can support other registry types like S3 by modifying the job image and pre-installing plugins.

## Ignoring Charts

You may want to only run charts on certain clusters. You can do this by templating the CRD and setting the `ignore:` field to true as per below.

```
apiVersion: helm.kubedex.com/v1
kind: HelmChart
metadata:
  name: myprivatechartname
  namespace: default
spec:
  chart: myprivatechartname
  repo: https://user:password@charts.example.com
  version: 1.0.0
  targetNamespace: default
  ignore: true
```

The helm-controller will now ignore this chart CRD. Be aware that this simply ignores the chart. The controller will not uninstall the chart.

You will need to delete the CRD for the controller to manage the delete.

# Installation Lifecycle

The helm-controller manifests should be deployed to the Kubernetes cluster using kubectl.

* A single CRD per Helm Chart
* The helm-controller triggers a Kubernetes job when a chart CRD is changed
* Each Kubernetes job executes the upgrade logic for the Helm Chart
* When a chart CRD is deleted the helm-controller will remove all resources associated with it

## Helm Chart CRD's

Chart CRD's define which Helm Charts a cluster should be running. You can view all chart CRD's by executing the following command.

```
kubectl get helmcharts.helm.kubedex --all-namespaces
```

Or, to look at the contents of a single CRD you can use this command:

```
kubectl get helmchart.helm.kubedex kubernetes-dashboard -o yaml
```

For testing and playing around you can edit the chart CRD's directly to bump chart version or change values.

On change the helm-controller will immediately execute a Kubernetes job to apply the Helm Chart upgrade.

# Troubleshooting

Use standard kubectl commands to validate each stage has completed successfully.

* Check that the helm-controller is running and the logs are clean
* Check the contents of the chart CRD on the cluster
* Check the Kubernetes job logs for the chart you are troubleshooting

To fully reset a chart you can delete the CRD. Then wait for all resources to be removed. Then apply the CRD again.

To remove all charts from a cluster you can run:

```
kubectl delete helmcharts.helm.kubedex --all-namespaces --all
```

# Credits

Heavily inspired by the [Rancher Helm Controller](https://github.com/rancher/helm-controller).

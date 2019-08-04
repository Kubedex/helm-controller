package helmchart

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
	"sort"

	helmv1 "github.com/Kubedex/helm-controller/pkg/apis/helm/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_helmchart")

const helmChartFinalizer = "finalizer.helm.kubedex.com"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new HelmChart Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHelmChart{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("helmchart-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HelmChart
	err = c.Watch(&source.Kind{Type: &helmv1.HelmChart{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileHelmChart implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileHelmChart{}

// ReconcileHelmChart reconciles a HelmChart object
type ReconcileHelmChart struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a HelmChart object and makes changes based on the state read
// and what is in the HelmChart.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHelmChart) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling HelmChart")

	// Fetch the HelmChart instance
	chart := &helmv1.HelmChart{}
	err := r.client.Get(context.TODO(), request.NamespacedName, chart)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define service account
	sa := r.serviceAccount(chart)
	foundSA := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      sa.ObjectMeta.Name,
		Namespace: sa.ObjectMeta.Namespace}, foundSA)

	// check the job exist if not crete
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ServiceAccount", "SA.Namespace",
			sa.ObjectMeta.Namespace, "SA.Name", sa.ObjectMeta.Name, )

		// create a api request to create service account
		// if failed reconcile again
		err = r.client.Create(context.TODO(), sa)
		if err != nil {
			// requeue the job
			return reconcile.Result{}, err
		}
		// move to next step
	} else if err != nil {
		// if error anything else return with failure
		reqLogger.Error(err, "Failed to create ServiceAccount for helm deployment job")
		// requeue the job
		return reconcile.Result{}, err
	}

	// Define role binding
	rb := r.roleBinding(chart)
	foundRB := &rbacv1.ClusterRoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      rb.ObjectMeta.Name,
		Namespace: rb.ObjectMeta.Namespace}, foundRB)

	// check the job exist if not crete
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new RoleBinding", "RB.Namespace",
			rb.ObjectMeta.Namespace, "RB.Name", rb.ObjectMeta.Name)

		// create a api request to create RoleBinding
		// if failed reconcile again
		err = r.client.Create(context.TODO(), rb)
		if err != nil {
			// requeue the job
			return reconcile.Result{}, err
		}
		// move to next step
	} else if err != nil {
		// if error anything else return with failure
		reqLogger.Error(err, "Failed to create RoleBinding for helm deployment job")
		// requeue the job
		return reconcile.Result{}, err
	}

	// Define ConfigMap
	configMap := r.configMap(chart)
	// config map will exist only if value overrides are defined
	if configMap != nil {
		foundCM := &corev1.ConfigMap{}
		err = r.client.Get(context.TODO(), types.NamespacedName{
			Name:      configMap.ObjectMeta.Name,
			Namespace: configMap.ObjectMeta.Namespace}, foundCM)

		// check the job exist if not crete
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new ConfigMap", "CM.Namespace",
				configMap.ObjectMeta.Namespace, "CM.Name", configMap.ObjectMeta.Name)

			// create a api request to create ConfigMap
			// if failed reconcile again
			err = r.client.Create(context.TODO(), configMap)
			if err != nil {
				// requeue the job
				return reconcile.Result{}, err
			}
			// move to next step
		} else if err != nil {
			// if error anything else return with failure
			reqLogger.Error(err, "Failed to create ConfigMap for helm deployment job")
			// requeue the job
			return reconcile.Result{}, err
		}
	}

	// Define a new job object
	job := r.newJob(chart, "")
	// Check if this Job already exists
	found := &batchv1.Job{}
	err = r.client.Get(context.TODO(), types.NamespacedName{
		Name:      job.ObjectMeta.Name,
		Namespace: job.ObjectMeta.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Job", "Job.Namespace", job.ObjectMeta.Namespace, "Job.Name", job.ObjectMeta.Name)

		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}
		// the job is created, update the job info
		found = job
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Helm deployment job")
		return reconcile.Result{}, err
	}

	// Check if the HelmChart instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isHelmChartMarkedToBeDeleted := chart.GetDeletionTimestamp() != nil
	if isHelmChartMarkedToBeDeleted {
		if contains(chart.GetFinalizers(), helmChartFinalizer) {
			// Run finalization logic for helmchartFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeHelmChart(reqLogger, chart); err != nil {
				if err.Error() == "job update required" {
					reqLogger.Info("requeue for finalize")
					return reconcile.Result{Requeue: true}, nil
				}
				return reconcile.Result{}, err
			}

			// Remove helmchartFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			chart.SetFinalizers(remove(chart.GetFinalizers(), helmChartFinalizer))
			err := r.client.Update(context.TODO(), chart)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{Requeue: false}, nil
	}

	version := ""
	valueHash := ""
	for _, v := range (*found).Spec.Template.Spec.Containers[0].Env {
		if v.Name == "VERSION" {
			version = v.Value
		}

		if v.Name == "VALUES_HASH" {
			valueHash = v.Value
		}
	}

	chartHash := sha256.Sum256([]byte(chart.Spec.ValuesContent))
	// remove existing job if version or the value hash is different
	if version != chart.Spec.Version || valueHash != hex.EncodeToString(chartHash[:]) {
		reqLogger.Info("Removing job for helm update",
			"Job.Namespace", found.ObjectMeta.Namespace, "Job.Name", found.ObjectMeta.Name)
		// remove job before creating new
		err = r.client.Delete(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update helm update job",
				"Job.Namespace", found.ObjectMeta.Namespace, "Job.Name", found.ObjectMeta.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Add finalizer for this CR
	if !contains(chart.GetFinalizers(), helmChartFinalizer) {
		if err := r.addFinalizer(reqLogger, chart); err != nil {
			return reconcile.Result{}, err
		}
		// added finalizer and we are good
		return reconcile.Result{Requeue: false}, nil
	}

	// Job already exists - don't requeue
	reqLogger.Info("Skip reconcile: Job already exists",
		"Job.Namespace", found.ObjectMeta.Namespace, "Job.Name", found.ObjectMeta.Name)
	return reconcile.Result{Requeue: false}, nil
}

func (r *ReconcileHelmChart) newJob(chart *helmv1.HelmChart, action string) (*batchv1.Job) {
	oneThousand := int32(1000)
	valuesHash := sha256.Sum256([]byte(chart.Spec.ValuesContent))

	name := fmt.Sprintf("helm-%s", chart.Name)
	if len(action) > 0 {
		name = fmt.Sprintf("helm-%s-%s", action, chart.Name)
	}

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: chart.Namespace,
			Labels: map[string]string{
				"label": chart.Name,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &oneThousand,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": chart.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            "helm",
							Image:           getEnv("KLIPPER_IMAGE", "rancher/klipper-helm:v0.1.5", ""),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            args(chart),
							Env: []corev1.EnvVar{
								{
									Name:  "NAME",
									Value: chart.Name,
								},
								{
									Name:  "VERSION",
									Value: chart.Spec.Version,
								},
								{
									Name:  "REPO",
									Value: chart.Spec.Repo,
								},
								{
									Name:  "VALUES_HASH",
									Value: hex.EncodeToString(valuesHash[:]),
								},
							},
						},
					},
					ServiceAccountName: fmt.Sprintf("helm-%s", chart.Name),
				},
			},
		},
	}
	setProxyEnv(job)

	job.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "values",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("chart-values-%s", chart.Name),
					},
				},
			},
		},
	}

	job.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			MountPath: "/config",
			Name:      "values",
		},
	}

	// Set service account instance as the owner and controller
	_ = controllerutil.SetControllerReference(chart, job, r.scheme)
	return job
}

func (r *ReconcileHelmChart) configMap(chart *helmv1.HelmChart) *corev1.ConfigMap {
	if chart.Spec.ValuesContent == "" {
		return nil
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("chart-values-%s", chart.Name),
			Namespace: chart.Namespace,
		},
		Data: map[string]string{
			"values.yaml": chart.Spec.ValuesContent,
		},
	}
	// Set service account instance as the owner and controller
	_ = controllerutil.SetControllerReference(chart, cm, r.scheme)
	return cm
}

func (r *ReconcileHelmChart) roleBinding(chart *helmv1.HelmChart) *rbacv1.ClusterRoleBinding {
	rb := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("helm-%s-%s", chart.Namespace, chart.Name),
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      fmt.Sprintf("helm-%s", chart.Name),
				Kind:      "ServiceAccount",
				Namespace: chart.Namespace,
			},
		},
	}
	// Set service account instance as the owner and controller
	_ = controllerutil.SetControllerReference(chart, rb, r.scheme)
	return rb
}

func (r *ReconcileHelmChart) serviceAccount(chart *helmv1.HelmChart) *corev1.ServiceAccount {
	trueVal := true
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("helm-%s", chart.Name),
			Namespace: chart.Namespace,
		},
		AutomountServiceAccountToken: &trueVal,
	}
	// Set service account instance as the owner and controller
	_ = controllerutil.SetControllerReference(chart, sa, r.scheme)
	return sa
}

func args(chart *helmv1.HelmChart) []string {
	if chart.GetDeletionTimestamp() != nil {
		return []string{
			"delete",
			"--purge", chart.Name,
		}
	}

	spec := chart.Spec
	args := []string{
		"install",
		"--name", chart.Name,
		spec.Chart,
	}
	if spec.TargetNamespace != "" {
		args = append(args, "--namespace", spec.TargetNamespace)
	}
	if spec.Repo != "" {
		args = append(args, "--repo", spec.Repo)
	}
	if spec.Version != "" {
		args = append(args, "--version", spec.Version)
	}

	for _, k := range keys(spec.Set) {
		val := spec.Set[k]
		if val.StrVal != "" {
			args = append(args, "--set-string", fmt.Sprintf("%s=%s", k, val.StrVal))
		} else {
			args = append(args, "--set", fmt.Sprintf("%s=%d", k, val.IntVal))
		}
	}

	return args
}

func keys(val map[string]intstr.IntOrString) []string {
	var keys []string
	for k := range val {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func setProxyEnv(job *batchv1.Job) {
	proxySysEnv := []string{
		"all_proxy",
		"ALL_PROXY",
		"http_proxy",
		"HTTP_PROXY",
		"https_proxy",
		"HTTPS_PROXY",
		"no_proxy",
		"NO_PROXY",
	}
	for _, proxyEnv := range proxySysEnv {
		proxyEnvValue := os.Getenv(proxyEnv)
		if len(proxyEnvValue) == 0 {
			continue
		}
		envar := corev1.EnvVar{
			Name:  proxyEnv,
			Value: proxyEnvValue,
		}
		job.Spec.Template.Spec.Containers[0].Env = append(
			job.Spec.Template.Spec.Containers[0].Env,
			envar)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func getEnv(env string, def string, override string) string {
	// return override regardless
	if len(override) > 0 {
		return override
	}
	// lookup environments and if value not found return default else return value
	v, found := os.LookupEnv(env)
	if !found {
		return def
	}
	return v
}

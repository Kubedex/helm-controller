package helmchart

import (
	"context"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	helmv1 "github.com/Kubedex/helm-controller/pkg/apis/helm/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ge "errors"
)

func (r *ReconcileHelmChart) addFinalizer(reqLogger logr.Logger, m *helmv1.HelmChart) error {
	reqLogger.Info("Adding Finalizer for the helm chart")
	m.SetFinalizers(append(m.GetFinalizers(), helmChartFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update helm chart with finalizer")
		return err
	}
	return nil
}

// finalizeHelmChart create a helm destroy job to destroy all the resources created by helm
// once completed controller will take create of garbage collection of resources created by
// controller itself.
func (r *ReconcileHelmChart) finalizeHelmChart(reqLogger logr.Logger, m *helmv1.HelmChart) error {
	// create a delete job
	job := r.newJob(m, "delete")
	found := &batchv1.Job{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: job.ObjectMeta.Name, Namespace: job.ObjectMeta.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		err := r.client.Create(context.TODO(), job)
		if err != nil {
			reqLogger.Error(err, "Failed create helm destroy job",
				"Job.Namespace", job.ObjectMeta.Namespace, "Job.Name", job.ObjectMeta.Name)
			return err
		}
		// job is created re-queue again
		return ge.New("job update required")
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Helm deployment job")
		return err
	}

	status := (*found).Status
	// job already exist, check the job status
	if status.Failed >= 1 {
		// previous job has failed try deleting the job
		// then next reconcile can re-try
		err := r.client.Delete(context.TODO(), job)
		if err != nil {
			reqLogger.Error(err, "Failed delete helm destroy job",
				"Job.Namespace", job.ObjectMeta.Namespace, "Job.Name", job.ObjectMeta.Name)
			return err
		}
		return err
	} else if status.Succeeded >= 1 {
		reqLogger.Info("Successfully finalized helm chart")
		return nil
	}

	return ge.New("job update required")
}
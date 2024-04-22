package controller

import (
	"context"
	"fmt"
	"time"

	kbatch "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	windtunnelv1alpha1 "github.com/CarnegieMellon-PlantD/PlantD-operator/api/v1alpha1"
	"github.com/CarnegieMellon-PlantD/PlantD-operator/pkg/cost"
	"github.com/CarnegieMellon-PlantD/PlantD-operator/pkg/utils"
)

const (
	costExporterPollingInterval = 5 * time.Second
	costExporterInterval        = 8 * time.Hour
)

// CostExporterReconciler reconciles a CostExporter object
type CostExporterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=windtunnel.plantd.org,resources=costexporters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=windtunnel.plantd.org,resources=costexporters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=windtunnel.plantd.org,resources=costexporters/finalizers,verbs=update
//+kubebuilder:rbac:groups=windtunnel.plantd.org,resources=experiments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *CostExporterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the requested CostExporter
	costExporter := &windtunnelv1alpha1.CostExporter{}
	if err := r.Get(ctx, req.NamespacedName, costExporter); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Unable to fetch CostExporter")
		return ctrl.Result{}, err
	}

	if !costExporter.Status.IsRunning {
		// Check if the Job should be run now
		curTime := time.Now()
		if costExporter.Status.LastCompletionTime != nil && (curTime.Sub(costExporter.Status.LastCompletionTime.Time) < costExporterInterval) {
			logger.Info("CostExporter is not due for a Job yet")
			return ctrl.Result{RequeueAfter: costExporterInterval - curTime.Sub(costExporter.Status.LastCompletionTime.Time)}, nil
		}

		allExperiments := &windtunnelv1alpha1.ExperimentList{}
		if err := r.List(ctx, allExperiments); err != nil {
			logger.Error(err, "Cannot list Experiments")
			return ctrl.Result{}, err
		}

		var allExperimentTags []*cost.ExperimentTags
		earliestTime := time.Now()
		for _, experiment := range allExperiments.Items {
			// Filter Experiments with CSP
			if experiment.Status.CloudProvider != costExporter.Spec.CloudServiceProvider {
				continue
			}

			// Calculate the earliest time of the Experiments
			var experimentTime time.Time
			if !experiment.CreationTimestamp.IsZero() {
				experimentTime = experiment.CreationTimestamp.Time
			}
			if !experiment.Spec.ScheduledTime.IsZero() && experiment.Spec.ScheduledTime.Time.Before(experiment.CreationTimestamp.Time) {
				experimentTime = experiment.Spec.ScheduledTime.Time
			}

			// Check if this Experiment's time is earlier than the current earliest time
			if experimentTime.Before(earliestTime) {
				earliestTime = experimentTime
			}

			// Extract the Name from the Experiment's metadata
			name := experiment.Namespace + "/" + experiment.Name
			// Extract the Tags from the Experiment's status field
			tagMap := experiment.Status.Tags

			// Create a list of Tag pairs for the current experiment's tags
			var tagList []*cost.Tag
			for key, value := range tagMap {
				tagList = append(tagList, &cost.Tag{Key: key, Value: value})
			}
			// Create an ExperimentTags object for the current experiment
			experimentTags := &cost.ExperimentTags{
				Name: name,
				Tags: tagList,
			}

			// Append the object to the list of all Experiments' tags
			allExperimentTags = append(allExperimentTags, experimentTags)
		}

		// Delete the previous Job if it exists
		oldJob := &kbatch.Job{}
		oldJobName := types.NamespacedName{
			Namespace: costExporter.Namespace,
			Name:      utils.GetCostExporterJobName(costExporter.Name),
		}
		if err := r.Get(ctx, oldJobName, oldJob); err == nil {
			if err := r.Delete(ctx, oldJob); err != nil {
				logger.Error(err, "Cannot delete old cost exporter Job")
				return ctrl.Result{}, err
			}
		}

		// Create the Job
		job, err := cost.CreateCostExporterJob(costExporter, allExperimentTags, earliestTime)
		if err != nil {
			logger.Error(err, "Cannot create manifest for cost exporter Job")
			return ctrl.Result{}, err
		}
		if err := ctrl.SetControllerReference(costExporter, job, r.Scheme); err != nil {
			logger.Error(err, "Cannot set controller reference for cost exporter Job")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, job); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Cannot create cost exporter Job")
			return ctrl.Result{}, err
		} else if err == nil {
			logger.Info("Created cost exporter Job")
		}

		costExporter.Status.IsRunning = true
		if err := r.Status().Update(ctx, costExporter); err != nil {
			logger.Error(err, "Cannot update the status")
			return ctrl.Result{}, err
		}
	} else {
		// Check the Job status
		job := &kbatch.Job{}
		jobName := types.NamespacedName{
			Namespace: costExporter.Namespace,
			Name:      utils.GetCostExporterJobName(costExporter.Name),
		}
		if err := r.Get(ctx, jobName, job); err != nil {
			logger.Error(err, fmt.Sprintf("Lost Job \"%s\"", jobName))
			return ctrl.Result{}, err
		}

		jobFinished, jobConditionType := isJobFinished(job)
		if jobFinished {
			switch jobConditionType {
			case kbatch.JobComplete:
				logger.Info("Cost exporter Job completed")
				costExporter.Status.LastCompletionTime = &metav1.Time{Time: time.Now()}

			case kbatch.JobFailed:
				logger.Info("Cost exporter Job failed")
			}

			costExporter.Status.IsRunning = false
			if err := r.Status().Update(ctx, costExporter); err != nil {
				logger.Error(err, "Cannot update the status")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: costExporterPollingInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CostExporterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&windtunnelv1alpha1.CostExporter{}).
		Complete(r)
}

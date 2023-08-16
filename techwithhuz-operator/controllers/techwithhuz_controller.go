/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	//"os"
	//"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/techwithhuz/techwithhuz-operator/api/v1alpha1"
)

const techwithhuzFinalizer = "cache.example.com/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailableTechwithhuz represents the status of the Deployment reconciliation
	typeAvailableTechwithhuz = "Available"
	// typeDegradedTechwithhuz represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	typeDegradedTechwithhuz = "Degraded"
)

// TechwithhuzReconciler reconciles a Techwithhuz object
type TechwithhuzReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=cache.techwithhuz.com,resources=techwithhuzs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.techwithhuz.com,resources=techwithhuzs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.techwithhuz.com,resources=techwithhuzs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Techwithhuz object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TechwithhuzReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	// Fetch the Techwithhuz instance
	// The purpose is check if the Custom Resource for the Kind Techwithhuz
	// is applied on the cluster if not we return nil to stop the reconciliation
	techwithhuz := &cachev1alpha1.Techwithhuz{}
	err := r.Get(ctx, req.NamespacedName, techwithhuz)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("techwithhuz resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get techwithhuz")
		return ctrl.Result{}, err
	}
	// Let's just set the status as Unknown when no status are available
	if techwithhuz.Status.Conditions == nil || len(techwithhuz.Status.Conditions) == 0 {
		meta.SetStatusCondition(&techwithhuz.Status.Conditions, metav1.Condition{Type: typeAvailableTechwithhuz, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, techwithhuz); err != nil {
			log.Error(err, "Failed to update techwithhuz status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the techwithhuz Custom Resource after update the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raise the issue "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, techwithhuz); err != nil {
			log.Error(err, "Failed to re-fetch techwithhuz")
			return ctrl.Result{}, err
		}
	}
	// Let's add a finalizer. Then, we can define some operations which should
	// occurs before the custom resource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(techwithhuz, techwithhuzFinalizer) {
		log.Info("Adding Finalizer for techwithhuz")
		if ok := controllerutil.AddFinalizer(techwithhuz, techwithhuzFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, techwithhuz); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}
	// Check if the Techwithhuz instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isTechwithhuzMarkedToBeDeleted := techwithhuz.GetDeletionTimestamp() != nil
	if isTechwithhuzMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(techwithhuz, techwithhuzFinalizer) {
			log.Info("Performing Finalizer Operations for techwithhuz before delete CR")

			// Let's add here an status "Downgrade" to define that this resource begin its process to be terminated.
			meta.SetStatusCondition(&techwithhuz.Status.Conditions, metav1.Condition{Type: typeDegradedTechwithhuz,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", techwithhuz.Name)})

			if err := r.Status().Update(ctx, techwithhuz); err != nil {
				log.Error(err, "Failed to update techwithhuz status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before remove the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.doFinalizerOperationsForTechwithhuz(techwithhuz)

			// Re-fetch the techwithhuz Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, techwithhuz); err != nil {
				log.Error(err, "Failed to re-fetch techwithhuz")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&techwithhuz.Status.Conditions, metav1.Condition{Type: typeDegradedTechwithhuz,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", techwithhuz.Name)})

			if err := r.Status().Update(ctx, techwithhuz); err != nil {
				log.Error(err, "Failed to update Techwithhuz status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for Techwithhuz after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(techwithhuz, techwithhuzFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for techwithhuz")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, techwithhuz); err != nil {
				log.Error(err, "Failed to remove finalizer for techwithhuz")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: techwithhuz.Name, Namespace: techwithhuz.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForTechwithhuz(techwithhuz)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for techwithhuz")

			// The following implementation will update the status
			meta.SetStatusCondition(&techwithhuz.Status.Conditions, metav1.Condition{Type: typeAvailableTechwithhuz,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", techwithhuz.Name, err)})

			if err := r.Status().Update(ctx, techwithhuz); err != nil {
				log.Error(err, "Failed to update techwithhuz status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// The CRD API is defining that the techwithhuz type, have a TechwithhuzSpec.Size field
	// to set the quantity of Deployment instances is the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	size := techwithhuz.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			// Re-fetch the techwithhuz Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, techwithhuz); err != nil {
				log.Error(err, "Failed to re-fetch techwithhuz")
				return ctrl.Result{}, err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&techwithhuz.Status.Conditions, metav1.Condition{Type: typeAvailableTechwithhuz,
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", techwithhuz.Name, err)})

			if err := r.Status().Update(ctx, techwithhuz); err != nil {
				log.Error(err, "Failed to update techwithhuz status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource before
		// update. Also, it will help ensure the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}

	// The following implementation will update the status
	meta.SetStatusCondition(&techwithhuz.Status.Conditions, metav1.Condition{Type: typeAvailableTechwithhuz,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", techwithhuz.Name, size)})

	if err := r.Status().Update(ctx, techwithhuz); err != nil {
		log.Error(err, "Failed to update techwithhuz status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// finalizeTechwithhuz will perform the required operations before delete the CR.
func (r *TechwithhuzReconciler) doFinalizerOperationsForTechwithhuz(cr *cachev1alpha1.Techwithhuz) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// Note: It is not recommended to use finalizers with the purpose of delete resources which are
	// created and managed in the reconciliation. These ones, such as the Deployment created on this reconcile,
	// are defined as depended of the custom resource. See that we use the method ctrl.SetControllerReference.
	// to set the ownerRef which means that the Deployment will be deleted by the Kubernetes API.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/

	// The following implementation will raise an event
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

// deploymentForTechwithhuz returns a Techwithhuz Deployment object
func (r *TechwithhuzReconciler) deploymentForTechwithhuz(techwithhuz *cachev1alpha1.Techwithhuz) (*appsv1.Deployment, error) {
	replicas := techwithhuz.Spec.Size
	ls := labelsForTechwithhuz(techwithhuz.Name)

	// Get the Operand image
	image := "nginx:latest"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      techwithhuz.Name,
			Namespace: techwithhuz.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "techwithhuz",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser:                &[]int64{1001}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: techwithhuz.Spec.ContainerPort,
							Name:          "techwithhuz",
						}},
						Command: []string{"sleep", "1000s"},
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(techwithhuz, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

// labelsForTechwithhuz returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsForTechwithhuz(name string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": "Techwithhuz",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/part-of":    "techwithhuz-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TechwithhuzReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Techwithhuz{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

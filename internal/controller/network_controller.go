/*
Copyright 2024.

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

package controller

import (
	"context"
	"github.com/go-logr/logr"
	kettlev1alpha1 "github.com/jdambly/kettle/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NetworkReconciler reconciles a Network object
type NetworkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=networking.kettle.io,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.kettle.io,resources=networks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.kettle.io,resources=networks/finalizers,verbs=update

// Reconcile handles the reconciliation of Network resources
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Todo move this out of the reconcile function, I'm just not sure where to put it yet
	r.logger.Info("Reconciling Network")

	// Fetch the Network instance
	network := &kettlev1alpha1.Network{}
	err := r.Get(ctx, req.NamespacedName, network)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected
			// this will trigger on a delete
			// Todo: should we do some cleanup here?
			r.logger.Info("Network resource not found. Ignoring since object must be deleted")
			// this is considered a successful reconciliation
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Network")
		// return the error so that the controller can retry the failed request at the next notification
		return ctrl.Result{}, err
	}

	// Initialize the Network
	if network.IsConditionPresentAndEqual(kettlev1alpha1.ConditionInitialized, metav1.ConditionTrue) {
		r.logger.Info("Network already initialized skipping...")
	} else {
		r.logger.Info("Network not initialized, initializing...")
		r.Initialize(network)
	}

	// Update the status of the Network
	err = r.Status().Update(ctx, network)
	if err != nil {
		r.logger.Error(err, "Failed to update Network status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.logger = log.FromContext(context.Background()).WithCallDepth(3)
	// create a predicate to only trigger the controller when the status is updated with changes to allocated IPs
	// this is to avoid unnecessary reconciliations
	statusUpdatePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*kettlev1alpha1.Network)
			newObj := e.ObjectNew.(*kettlev1alpha1.Network)
			return oldObj.ShouldReconcile(newObj)
		},
	}
	// predicate is used here to make sure the controller triggers when the status is updated
	// it should also trigger on create/update/delete events
	return ctrl.NewControllerManagedBy(mgr).
		For(&kettlev1alpha1.Network{}).
		WithEventFilter(statusUpdatePredicate).
		Complete(r)
}

// Initialize checks if the Network is initialized and if not allocates ip addresses and sets the Initialized condition
func (r *NetworkReconciler) Initialize(network *kettlev1alpha1.Network) {
	// Allocate IP addresses
	allocatableIPs, err := network.GetAllocatableIPs()
	if err != nil {
		r.logger.Error(err, "Failed to allocate IP addresses")
		// set the Initialized condition to False
		network.SetConditionInitialized(metav1.ConditionFalse)
	}

	// Update the Network status with the allocated IP addresses
	network.Status.FreeIPs = allocatableIPs
	network.Status.AssignedIPs = []kettlev1alpha1.AllocatedIP{}

	// Set the Initialized condition to True
	network.SetConditionInitialized(metav1.ConditionTrue)
}

// UpdateIPs updates the FreeIPs in the Network status
func (r *NetworkReconciler) UpdateIPs(network *kettlev1alpha1.Network) bool {
	r.logger.Info("Updating FreeIPs")
	updated := false

	// Create a map of assigned IPs for faster lookup
	assignedIPMap := make(map[string]bool)
	for _, assignedIP := range network.Status.AssignedIPs {
		assignedIPMap[assignedIP.IP] = true
	}

	// Create a new list for FreeIPs, keeping only the IPs that are still available
	var updatedFreeIPs []string
	for _, freeIP := range network.Status.FreeIPs {
		if !assignedIPMap[freeIP] {
			updatedFreeIPs = append(updatedFreeIPs, freeIP)
		} else {
			updated = true
		}
	}

	// If updates occurred, modify the network status but keep existing fields
	if updated {
		network.Status.FreeIPs = updatedFreeIPs

		// Set a condition to indicate that the FreeIPs have been updated
		network.SetConditionFreeIPsUpdated(metav1.ConditionTrue)
	}

	return updated
}

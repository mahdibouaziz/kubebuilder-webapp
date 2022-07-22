/*
Copyright 2022.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webappv1 "github.com/mahdibouaziz/kubebuilder-webapp/api/v1"
)

// GuestbookReconciler reconciles a Guestbook object
type GuestbookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.example.com,resources=guestbooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.example.com,resources=guestbooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.example.com,resources=guestbooks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Guestbook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *GuestbookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	// Get the root object (GuestBook)
	var book webappv1.Guestbook
	if err := r.Get(ctx, req.NamespacedName, &book); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get the depending objects (Redis)
	var redis webappv1.Redis
	// create a NamespceName object (key) to use it to get the redis object
	redisName := client.ObjectKey{Namespace: req.Name, Name: book.Spec.RedisName}
	if err := r.Get(ctx, redisName, &redis); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// use the informatin from the Redis status and our guestbook spec
	// to contruct the deployment to run our guestbook frontend
	deployment, err := r.desiredDeployment(book, redis)
	if err != nil {
		return ctrl.Result{}, err
	}

	// use the information from the guestbook spec to construct a service that exposes that deployment
	svc, err := r.desiredService(book)
	if err != nil {
		return ctrl.Result{}, err
	}

	// submit those changes to the API Server
	applyOpts := []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("guestbook-controller"),
	}

	err = r.Patch(ctx, &deployment, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, &svc, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update the status
	book.Status.URL = urlForService(svc, book.Spec.Frontend.ServingPort)
	err = r.Status().Update(ctx, &book)
	if err != nil {
		return ctrl.Result{}, err
	}

	// return the result value
	log.Log.Info("reconciled guestbook")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GuestbookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Guestbook{}).
		Complete(r)
}

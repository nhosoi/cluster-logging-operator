package clusterlogging

import (
	"context"
	"time"

	loggingv1 "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/pkg/k8shandler"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// "sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_clusterlogging")

const (
	singletonName    = "instance"
	singletonMessage = "ClusterLogging is a singleton. Only an instance named 'instance' is allowed"
)

// Add creates a new ClusterLogging Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterLogging{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterlogging-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	/******** DID NOT WORK *********
	h := &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &loggingv1.ClusterLogging{},
		 }
	******** DID NOT WORK *********/

	/******** DID NOT WORK *********
	// We only care about a configmap source with a specific name/namespace,
	// so filter events before they are provided to the controller event handlers.
	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			b := handleClusterLogging(e.MetaNew)
			logrus.Infof("DBG_CL: Update '%v'", b)
			return b
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			b := handleClusterLogging(e.Meta)
			logrus.Infof("DBG_CL: Delete '%v'", b)
			return b
		},
		CreateFunc: func(e event.CreateEvent) bool {
			b := handleClusterLogging(e.Meta)
			logrus.Infof("DBG_CL: Create '%v'", b)
			return b
		},
		GenericFunc: func(e event.GenericEvent) bool {
			b := handleClusterLogging(e.Meta)
			logrus.Infof("DBG_CL: Generic '%v'", b)
			return b
		},
	}
	// Watch for changes to primary resource ClusterLogging
	err = c.Watch(&source.Kind{Type: &loggingv1.ClusterLogging{}}, &handler.EnqueueRequestForObject{}, pred)
	******** DID NOT WORK *********/
	err = c.Watch(&source.Kind{Type: &loggingv1.ClusterLogging{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

/******** DID NOT WORK *********
// handleClusterLogging returns true if meta namespace is "openshift-logging".
func handleClusterLogging(meta metav1.Object) bool {
	return meta.GetNamespace() == "openshift-logging"
}
******** DID NOT WORK *********/

var _ reconcile.Reconciler = &ReconcileClusterLogging{}

// ReconcileClusterLogging reconciles a ClusterLogging object
type ReconcileClusterLogging struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

var (
	reconcilePeriod = 30 * time.Second
	reconcileResult = reconcile.Result{RequeueAfter: reconcilePeriod}
)

// Reconcile reads that state of the cluster for a ClusterLogging object and makes changes based on the state read
// and what is in the ClusterLogging.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileClusterLogging) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logrus.Infof("DBG_CL: Clusterlogging reconcile request.Name: '%s'", request.Name)
	// Fetch the ClusterLogging instance
	instance := &loggingv1.ClusterLogging{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	if instance.Name != singletonName {
		// TODO: update status

		logrus.Infof("DBG_CL: Clusterlogging instance.Name != singletonName: '%s' != '%s'", instance.Name, singletonName)
		return reconcile.Result{}, nil
	}

	if instance.Spec.ManagementState == loggingv1.ManagementStateUnmanaged {
		return reconcile.Result{}, nil
	}

	if err = k8shandler.Reconcile(instance, r.client, false); err != nil {
		return reconcileResult, err
	}

	return reconcileResult, nil
}

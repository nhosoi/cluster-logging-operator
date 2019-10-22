package proxyconfig

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	loggingv1 "github.com/openshift/cluster-logging-operator/pkg/apis/logging/v1"
	// "github.com/openshift/cluster-logging-operator/pkg/k8shandler"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_proxyconfig")

const (
	// singletonName          = "instance"
	// singletonMessage       = "ClusterLogging is a singleton. Only an instance named 'instance' is allowed"
	proxyName              = "cluster"
	trustBundleConfigmapNS = "openshift-config"
)

// Add creates a new ClusterLogging Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	if err := configv1.Install(mgr.GetScheme()); err != nil {
		return &ReconcileProxyConfig{}
	}

	return &ReconcileProxyConfig{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("proxyconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// We only care about a configmap source with a specific name/namespace,
	// so filter events before they are provided to the controller event handlers.
	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			b := handleConfigMap(e.MetaNew)
			logrus.Infof("DBG_PX: Update '%v'", b)
			return b
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			b := handleConfigMap(e.Meta)
			logrus.Infof("DBG_PX: Delete '%v'", b)
			return b
		},
		CreateFunc: func(e event.CreateEvent) bool {
			b := handleConfigMap(e.Meta)
			logrus.Infof("DBG_PX: Create '%v'", b)
			return b
		},
		GenericFunc: func(e event.GenericEvent) bool {
			b := handleConfigMap(e.Meta)
			logrus.Infof("DBG_PX: Generic '%v'", b)
			return b
		},
	}

	/******** DID NOT WORK *********
	h := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &configv1.Proxy{},
	}
	******** DID NOT WORK *********/

	// Watch for changes to the additional trust bundle configmap.
	if err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, pred); err != nil {
		return err
	}

	// Watch for changes to the proxy resource.
	if err = c.Watch(&source.Kind{Type: &configv1.Proxy{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

// handleConfigMap returns true if meta namespace is "openshift-config".
// "openshift-config" is the namespace for one or more
// ConfigMaps that contain user provided trusted CA bundles.
func handleConfigMap(meta metav1.Object) bool {
	return meta.GetNamespace() == "openshift-config"
}

var _ reconcile.Reconciler = &ReconcileProxyConfig{}

// ReconcileProxyConfig reconciles a ClusterLogging object
type ReconcileProxyConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func proxyNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name: "cluster",
	}
}

// Reconcile reads that state of the cluster for a cluslter-scoped
// named "cluster" or a configmap object in namespace "openshift-config".
func (r *ReconcileProxyConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.NamespacedName == proxyNamespacedName() {
		proxyConfig := &configv1.Proxy{}
		logrus.Infof("DBG_PX: Reconciling proxy request.Name: '%s'", request.Name)
		if err := r.client.Get(context.TODO(), request.NamespacedName, proxyConfig); err != nil {
			if apierrors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Return and don't requeue
				logrus.Errorf("proxy not found; reconciliation will be skipped - request %v", request)
				return reconcile.Result{}, nil
			}
			// Error reading the object - requeue the request.
			return reconcile.Result{}, fmt.Errorf("failed to get proxy '%s': %v", request.Name, err)
		}

		if proxyConfig.Name != proxyName {
			logrus.Infof("DBG_PX: Reconciling proxy proxyConfig.Name != proxyName: '%s' != '%s'", proxyConfig.Name, proxyName)
			return reconcile.Result{}, nil
		}
	} else if request.Namespace == trustBundleConfigmapNS {
		trustBundle := &corev1.ConfigMap{}
		logrus.Infof("DBG_PX: Reconciling additional trust bundle configmap request.Namespace/request.Name: '%s/%s'", request.Namespace, request.Name)
		if err := r.client.Get(context.TODO(), request.NamespacedName, trustBundle); err != nil {
			if apierrors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Return and don't requeue
				logrus.Errorf("configmap not found; reconciliation will be skipped - request %v", request)
				return reconcile.Result{}, nil
			}
			// Error reading the object - requeue the request.
			return reconcile.Result{}, fmt.Errorf("failed to get configmap '%s': %v", request, err)
		}

		// Only proceed if request matches the configmap referenced by proxy trustedCA.
		if err := r.configMapIsProxyTrustedCA(trustBundle.Name); err != nil {
			logrus.Errorf("configmap '%s/%s' name differs from trustedCA of proxy '%s' or trustedCA not set; "+
				"reconciliation will be skipped", trustBundle.Namespace, trustBundle.Name, proxyName)
			return reconcile.Result{}, nil
		}
	} else {
		return reconcile.Result{}, nil
	}

	/******** DID NOT WORK *********
	// Fetch the ClusterLogging instance
	instance := &loggingv1.ClusterLogging{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Spec.ManagementState == loggingv1.ManagementStateUnmanaged {
		return reconcile.Result{}, nil
	}

	logrus.Infof("DBG_PX: Reconciling collection pods")
	if err := k8shandler.Reconcile(instance, r.client, true); err != nil {
		return reconcile.Result{}, err
	}
	******** DID NOT WORK *********/

	return reconcile.Result{}, nil
}

// configMapIsProxyTrustedCA returns an error if cfgMapName does not match the
// ConfigMap name referenced by proxy "cluster" trustedCA.
func (r *ReconcileProxyConfig) configMapIsProxyTrustedCA(cfgMapName string) error {
	proxyConfig := &configv1.Proxy{}
	err := r.client.Get(context.TODO(), proxyNamespacedName(), proxyConfig)
	if err != nil {
		return fmt.Errorf("failed to get proxy '%s': %v", proxyName, err)
	}

	if proxyConfig.Spec.TrustedCA.Name != cfgMapName {
		return fmt.Errorf("configmap name '%s' does not match proxy trustedCA name '%s'", cfgMapName,
			proxyConfig.Spec.TrustedCA.Name)
	}

	return nil
}

package node

import (
	"context"
	"encoding/json"
	"fmt"

	corev1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_node")

const extendedResourceName string = "eks~1attachments~1EBS"

const filterLabel string = "beta.kubernetes.io/instance-type"

var volumesPerNodeType = map[string]string{
	"m5a.2xlarge": "20",
}

type jsonPayloadData struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNode{client: mgr.GetClient(), scheme: mgr.GetScheme(), cfg: mgr.GetConfig()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("node-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Node
	err = c.Watch(&source.Kind{Type: &corev1api.Node{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNode{}

// ReconcileNode reconciles a Node object
type ReconcileNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	cfg    *rest.Config
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Name", request.Name)
	reqLogger.Info("Reconciling Node")

	// Fetch the Node instance
	instance := &corev1api.Node{}
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

	nodeType, typeFound := instance.Labels[filterLabel]
	extResourceValue, resValueFound := instance.Status.Capacity[corev1api.ResourceName(extendedResourceName)]
	if typeFound {
		volumes := volumesPerNodeType[nodeType]
		if resValueFound || extResourceValue.String() != volumes {
			reqLogger.Info("Reconcile: node capacity must be set to", "NodeType", nodeType, "MaxVolumes", volumes)

			jsonData := make([]jsonPayloadData, 1)
			jsonData[0].Op = "add"
			jsonData[0].Path = fmt.Sprintf("/status/capacity/%s", extendedResourceName)
			jsonData[0].Value = volumes

			jsonStr, err := json.Marshal(jsonData)
			if err != nil {
				reqLogger.Info("Failed to marshal PayloadData")
				return reconcile.Result{}, err
			}

			clientset, err := kubernetes.NewForConfig(r.cfg)
			if err != nil {
				reqLogger.Info("Failed to create clientset")
				return reconcile.Result{}, err
			}

			res, err := clientset.
				CoreV1().
				Nodes().
				Patch(instance.Name, types.JSONPatchType, jsonStr, "status")
			if err != nil {
				reqLogger.Info("Failed to patch the node", "NodeName", instance.Name)
				return reconcile.Result{}, err
			}

			reqLogger.Info("Reconcile: Result", "NewNodeDefinition", res)
		} else {
			reqLogger.Info("External Resource Value is already set", "extResourceValue", extResourceValue.String())
			return reconcile.Result{}, nil
		}
	} else {
		reqLogger.Info("NodeType not found - skipping", "nodeType", nodeType)
		return reconcile.Result{}, nil
	}

	reqLogger.Info("Finished reconcile")
	return reconcile.Result{}, nil
}

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

const extendedResourceName string = "eks.ebsnodeoperator~1attachments"

//const filterLabel string = "beta.kubernetes.io/instance-type"

var (
	log          = logf.Log.WithName("controller_node")
	filterLabels = []string{
		"node.kubernetes.io/instance-type",
		"beta.kubernetes.io/instance-type",
	}
)

// Based on calculation from: https://github.com/kubernetes/kubernetes/issues/80967
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html
// A1, C5, C5d, C5n, I3en, M5, M5a, M5ad, M5d, p3dn.24xlarge, R5, R5a, R5ad, R5d, T3, T3a, and z1d <= 28
// 28 - 1 (root volume) - 110/interface capacity (num of interfaces) - number of NVMe volumes
var volumesPerNodeType = map[string]string{
	"a1.medium":     "25",
	"a1.large":      "24",
	"a1.xlarge":     "23",
	"a1.2xlarge":    "23",
	"a1.4xlarge":    "23",
	"a1.metal":      "23",
	"c1.medium":     "25",
	"c1.xlarge":     "23",
	"c3.large":      "24",
	"c3.xlarge":     "23",
	"c3.2xlarge":    "23",
	"c3.4xlarge":    "23",
	"c3.8xlarge":    "23",
	"c4.large":      "24",
	"c4.xlarge":     "23",
	"c4.2xlarge":    "23",
	"c4.4xlarge":    "23",
	"c4.8xlarge":    "23",
	"c5.large":      "24",
	"c5.xlarge":     "23",
	"c5.2xlarge":    "23",
	"c5.4xlarge":    "23",
	"c5.9xlarge":    "23",
	"c5.12xlarge":   "23",
	"c5.18xlarge":   "24",
	"c5.24xlarge":   "24",
	"c5.metal":      "24",
	"c5d.large":     "23",
	"c5d.xlarge":    "22",
	"c5d.2xlarge":   "22",
	"c5d.4xlarge":   "22",
	"c5d.9xlarge":   "22",
	"c5d.12xlarge":  "21",
	"c5d.18xlarge":  "22",
	"c5d.24xlarge":  "20",
	"c5d.metal":     "20",
	"c5n.large":     "24",
	"c5n.xlarge":    "23",
	"c5n.2xlarge":   "23",
	"c5n.4xlarge":   "23",
	"c5n.9xlarge":   "23",
	"c5n.18xlarge":  "24",
	"c5n.metal":     "24",
	"cc2.8xlarge":   "23",
	"cr1.8xlarge":   "23",
	"d2.xlarge":     "23",
	"d2.2xlarge":    "23",
	"d2.4xlarge":    "23",
	"d2.8xlarge":    "23",
	"f1.2xlarge":    "23",
	"f1.4xlarge":    "23",
	"f1.16xlarge":   "24",
	"g2.2xlarge":    "23",
	"g2.8xlarge":    "23",
	"g3s.xlarge":    "23",
	"g3.4xlarge":    "23",
	"g3.8xlarge":    "23",
	"g3.16xlarge":   "24",
	"g4dn.xlarge":   "24",
	"g4dn.2xlarge":  "24",
	"g4dn.4xlarge":  "24",
	"g4dn.8xlarge":  "23",
	"g4dn.12xlarge": "23",
	"g4dn.16xlarge": "23",
	"h1.2xlarge":    "23",
	"h1.4xlarge":    "23",
	"h1.8xlarge":    "23",
	"h1.16xlarge":   "24",
	"hs1.8xlarge":   "23",
	"i2.xlarge":     "23",
	"i2.2xlarge":    "23",
	"i2.4xlarge":    "23",
	"i2.8xlarge":    "23",
	"i3.large":      "23",
	"i3.xlarge":     "22",
	"i3.2xlarge":    "22",
	"i3.4xlarge":    "21",
	"i3.8xlarge":    "19",
	"i3.16xlarge":   "16",
	"i3.metal":      "16",
	"i3en.large":    "24",
	"i3en.xlarge":   "23",
	"i3en.2xlarge":  "23",
	"i3en.3xlarge":  "23",
	"i3en.6xlarge":  "23",
	"i3en.12xlarge": "23",
	"i3en.24xlarge": "24",
	"i3en.metal":    "24",
	"inf1.xlarge":   "23",
	"inf1.2xlarge":  "23",
	"inf1.6xlarge":  "23",
	"inf1.24xlarge": "23",
	"m1.small":      "25",
	"m1.medium":     "25",
	"m1.large":      "24",
	"m1.xlarge":     "23",
	"m2.xlarge":     "23",
	"m2.2xlarge":    "23",
	"m2.4xlarge":    "23",
	"m3.medium":     "25",
	"m3.large":      "24",
	"m3.xlarge":     "23",
	"m3.2xlarge":    "23",
	"m4.large":      "25",
	"m4.xlarge":     "23",
	"m4.2xlarge":    "23",
	"m4.4xlarge":    "23",
	"m4.10xlarge":   "23",
	"m4.16xlarge":   "23",
	"m5.large":      "24",
	"m5.xlarge":     "23",
	"m5.2xlarge":    "23",
	"m5.4xlarge":    "23",
	"m5.8xlarge":    "23",
	"m5.12xlarge":   "23",
	"m5.16xlarge":   "24",
	"m5.24xlarge":   "24",
	"m5.metal":      "24",
	"m5a.large":     "24",
	"m5a.xlarge":    "23",
	"m5a.2xlarge":   "23",
	"m5a.4xlarge":   "23",
	"m5a.8xlarge":   "23",
	"m5a.12xlarge":  "23",
	"m5a.16xlarge":  "24",
	"m5a.24xlarge":  "24",
	"m5ad.large":    "23",
	"m5ad.xlarge":   "22",
	"m5ad.2xlarge":  "22",
	"m5ad.4xlarge":  "21",
	"m5ad.8xlarge":  "21",
	"m5ad.12xlarge": "21",
	"m5ad.16xlarge": "20",
	"m5ad.24xlarge": "20",
	"m5d.large":     "23",
	"m5d.xlarge":    "22",
	"m5d.2xlarge":   "22",
	"m5d.4xlarge":   "21",
	"m5d.8xlarge":   "21",
	"m5d.12xlarge":  "21",
	"m5d.16xlarge":  "20",
	"m5d.24xlarge":  "20",
	"m5d.metal":     "20",
	"m5dn.large":    "23",
	"m5dn.xlarge":   "22",
	"m5dn.2xlarge":  "22",
	"m5dn.4xlarge":  "21",
	"m5dn.8xlarge":  "21",
	"m5dn.12xlarge": "21",
	"m5dn.16xlarge": "20",
	"m5dn.24xlarge": "20",
	"m5n.large":     "24",
	"m5n.xlarge":    "23",
	"m5n.2xlarge":   "23",
	"m5n.4xlarge":   "23",
	"m5n.8xlarge":   "23",
	"m5n.12xlarge":  "23",
	"m5n.16xlarge":  "24",
	"m5n.24xlarge":  "24",
	"p2.xlarge":     "23",
	"p2.8xlarge":    "23",
	"p2.16xlarge":   "23",
	"p3.2xlarge":    "23",
	"p3.8xlarge":    "23",
	"p3.16xlarge":   "23",
	"p3dn.24xlarge": "24",
	"r3.large":      "24",
	"r3.xlarge":     "23",
	"r3.2xlarge":    "23",
	"r3.4xlarge":    "23",
	"r3.8xlarge":    "23",
	"r4.large":      "24",
	"r4.xlarge":     "23",
	"r4.2xlarge":    "23",
	"r4.4xlarge":    "23",
	"r4.8xlarge":    "23",
	"r4.16xlarge":   "24",
	"r5.large":      "24",
	"r5.xlarge":     "23",
	"r5.2xlarge":    "23",
	"r5.4xlarge":    "23",
	"r5.8xlarge":    "23",
	"r5.12xlarge":   "23",
	"r5.16xlarge":   "24",
	"r5.24xlarge":   "24",
	"r5.metal":      "24",
	"r5a.large":     "24",
	"r5a.xlarge":    "23",
	"r5a.2xlarge":   "23",
	"r5a.4xlarge":   "23",
	"r5a.8xlarge":   "23",
	"r5a.12xlarge":  "23",
	"r5a.16xlarge":  "24",
	"r5a.24xlarge":  "24",
	"r5ad.large":    "23",
	"r5ad.xlarge":   "22",
	"r5ad.2xlarge":  "22",
	"r5ad.4xlarge":  "21",
	"r5ad.8xlarge":  "21",
	"r5ad.12xlarge": "21",
	"r5ad.16xlarge": "20",
	"r5ad.24xlarge": "20",
	"r5d.large":     "23",
	"r5d.xlarge":    "22",
	"r5d.2xlarge":   "22",
	"r5d.4xlarge":   "21",
	"r5d.8xlarge":   "21",
	"r5d.12xlarge":  "21",
	"r5d.16xlarge":  "20",
	"r5d.24xlarge":  "20",
	"r5d.metal":     "20",
	"r5dn.large":    "23",
	"r5dn.xlarge":   "22",
	"r5dn.2xlarge":  "22",
	"r5dn.4xlarge":  "21",
	"r5dn.8xlarge":  "21",
	"r5dn.12xlarge": "21",
	"r5dn.16xlarge": "20",
	"r5dn.24xlarge": "20",
	"r5n.large":     "24",
	"r5n.xlarge":    "23",
	"r5n.2xlarge":   "23",
	"r5n.4xlarge":   "23",
	"r5n.8xlarge":   "23",
	"r5n.12xlarge":  "23",
	"r5n.16xlarge":  "24",
	"r5n.24xlarge":  "24",
	"t1.micro":      "25",
	"t2.nano":       "25",
	"t2.micro":      "25",
	"t2.small":      "24",
	"t2.medium":     "24",
	"t2.large":      "24",
	"t2.xlarge":     "24",
	"t2.2xlarge":    "24",
	"t3.nano":       "25",
	"t3.micro":      "25",
	"t3.small":      "24",
	"t3.medium":     "24",
	"t3.large":      "24",
	"t3.xlarge":     "23",
	"t3.2xlarge":    "23",
	"t3a.nano":      "25",
	"t3a.micro":     "25",
	"t3a.small":     "25",
	"t3a.medium":    "24",
	"t3a.large":     "24",
	"t3a.xlarge":    "23",
	"t3a.2xlarge":   "23",
	"u-6tb1.metal":  "23",
	"u-9tb1.metal":  "23",
	"u-12tb1.metal": "23",
	"u-18tb1.metal": "24",
	"u-24tb1.metal": "24",
	"x1.16xlarge":   "23",
	"x1.32xlarge":   "23",
	"x1e.xlarge":    "24",
	"x1e.2xlarge":   "23",
	"x1e.4xlarge":   "23",
	"x1e.8xlarge":   "23",
	"x1e.16xlarge":  "23",
	"x1e.32xlarge":  "23",
	"z1d.large":     "23",
	"z1d.xlarge":    "22",
	"z1d.2xlarge":   "22",
	"z1d.3xlarge":   "22",
	"z1d.6xlarge":   "22",
	"z1d.12xlarge":  "22",
	"z1d.metal":     "22",
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

	typeFound := false
	nodeType := "---"
	for _, filterLabel := range filterLabels {
		nodeType, typeFound = instance.Labels[filterLabel]
		if typeFound {
			break
		}
	}

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

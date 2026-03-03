package main

import (
    "context"
    "os"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
    intendedLabel = "egress-gateway-intended"
    activeLabel   = "egress-gateway-active"
)

type NodeReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var node corev1.Node

    // Fetch latest Node state from API (via cache)
    if err := r.Get(ctx, types.NamespacedName{Name: req.Name}, &node); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Check if node is intended gateway
    intended := node.Labels[intendedLabel] == "true"

    // Determine Ready condition
    ready := isNodeReady(&node)

    active := node.Labels[activeLabel] == "true"

    // If not intended, ensure active label removed
    if !intended && active {
        delete(node.Labels, activeLabel)
        return ctrl.Result{}, r.Update(ctx, &node)
    }

    // If intended + ready → must be active
    if intended && ready && !active {
        if node.Labels == nil {
            node.Labels = map[string]string{}
        }
        node.Labels[activeLabel] = "true"
        return ctrl.Result{}, r.Update(ctx, &node)
    }

    // If intended but not ready → remove active
    if intended && !ready && active {
        delete(node.Labels, activeLabel)
        return ctrl.Result{}, r.Update(ctx, &node)
    }

    return ctrl.Result{}, nil
}

func isNodeReady(node *corev1.Node) bool {
    for _, cond := range node.Status.Conditions {
        if cond.Type == corev1.NodeReady {
            return cond.Status == corev1.ConditionTrue
        }
    }
    return false
}

func main() {
    scheme := runtime.NewScheme()
    _ = corev1.AddToScheme(scheme)

    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        Scheme: scheme,
        LeaderElection: true,
        LeaderElectionID: "node-label-controller",
    })
    if err != nil {
        os.Exit(1)
    }

    reconciler := &NodeReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }

    if err := ctrl.NewControllerManagedBy(mgr).
        For(&corev1.Node{}).
        Complete(reconciler); err != nil {
        os.Exit(1)
    }

    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        os.Exit(1)
    }
}

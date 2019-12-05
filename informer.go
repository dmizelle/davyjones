package main

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/util/taints"
)

type watcher struct {
	nodes        []string
	restarter    chan struct{}
	clientSet    *kubernetes.Clientset
	nodeInformer *cache.Controller
	config       *Config
}

func NewWatcher(clientSet *kubernetes.Clientset, cfg *Config) *watcher {
	return &watcher{
		clientSet: clientSet,
		config:    cfg,
	}
}

func (w *watcher) Run() {
	podInformer := w.NewPodInformer("kube-system")

	stop := make(chan struct{})

	podInformer.Run(stop)
}

func (w *watcher) NewPodInformer(namespace string) cache.Controller {
	podWatchList := cache.NewListWatchFromClient(
		w.clientSet.CoreV1().RESTClient(),
		"pods",
		namespace,
		fields.Everything(),
	)
	_, controller := cache.NewInformer(
		podWatchList,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				log.Infow("new pod detected",
					"podName", pod.Name,
					"namespace", namespace,
				)
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				log.Infow("pod has been deleted",
					"podName", pod.Name,
					"namespace", namespace,
				)
			},
			// TODO: Move this into its own function.
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldPod := oldObj.(*v1.Pod)
				newPod := newObj.(*v1.Pod)
				log.Infow("pod has changed status phase",
					"podName", newPod.Name,
					"namespace", namespace,
					"oldPhase", oldPod.Status.Phase,
					"newPhase", newPod.Status.Phase,
					"nodeName", newPod.Spec.NodeName,
				)
				owner := newPod.GetOwnerReferences()
				for _, ds := range cfg.DaemonSets {
					log.Debugw("looping through daemonset",
						"namespace", ds.Namespace,
						"name", ds.Name,
					)
					for _, ownerRef := range owner {
						log.Debugw("looping through owner",
							"podName", newPod.Name,
							"dsName", ownerRef.Name,
						)
						// OwnerReference does not expose namespace
						// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#OwnerReference
						if ds.Namespace == namespace && ownerRef.Kind == "DaemonSet" && ownerRef.Name == ds.Name {
							var err error
							if newPod.Status.Phase != "Running" {
								log.Debugw("tainting node",
									"podName", newPod.Name,
									"podPhase", newPod.Status.Phase,
									"nodeName", newPod.Spec.NodeName,
								)
								_, err = w.ChangeNodeTaint(newPod.Spec.NodeName, true)
							} else {
								log.Debugw("ensuring node is not tainted",
									"podName", newPod.Name,
									"podPhase", newPod.Status.Phase,
									"nodeName", newPod.Spec.NodeName,
								)
								_, err = w.ChangeNodeTaint(newPod.Spec.NodeName, false)
							}
							if err != nil {
								log.Errorw("unable to taint node",
									"podName", newPod.Name,
									"nodeName", newPod.Spec.NodeName,
									"error", err.Error(),
								)
							}
						}
					}
				}
			},
		},
	)
	return controller
}

func (w *watcher) ChangeNodeTaint(nodeName string, wantTaint bool) (*v1.Node, error) {
	node, err := w.clientSet.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var taintEffect v1.TaintEffect
	if w.config.Evict {
		taintEffect = v1.TaintEffectNoExecute
	} else {
		taintEffect = v1.TaintEffectNoSchedule
	}

	if wantTaint {
		taint := &v1.Taint{
			Key:    "taint.davyjones.github.com",
			Value:  "true",
			Effect: taintEffect,
		}
		node, _, err := taints.AddOrUpdateTaint(node, taint)
		if err != nil {
			return node, err
		}
		_, err = w.clientSet.CoreV1().Nodes().Update(node)
		return node, err
	}

	taintList := node.Spec.Taints
	taintList, _ = taints.DeleteTaintsByKey(taintList, "taint.davyjones.github.com")

	node.Spec.Taints = taintList
	_, err = w.clientSet.CoreV1().Nodes().Update(node)

	return node, err
}

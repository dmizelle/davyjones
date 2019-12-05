# WARNING -- THIS IS NOT PRODUCTION READY. NO TESTS. YOU'LL SINK.

# Davy Jones

This is an "operator" (not really, since there are no CRDs) that will watch for specified DaemonSets and ~~sink~~ _taint_ ~~ships~~ nodes ~~to the bottom of the sea~~ to prevent other deployments from running on them.

This project is _very_ similar to [uswitch/nidhogg](https://github.com/uswitch/nidhogg), but I wanted to try writing it myself, and without using (the awesome) [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project (that nidhogg uses.)

# Configuration Format

Configuration is provided in YAML and by default should exist at `/davyjones.yaml`.

Recommendation is to store it as a [ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#create-a-configmap) and [mount it at](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#add-configmap-data-to-a-specific-path-in-the-volume) the default location.

```yaml
---
nodeLabels:
  - label: node-role.kubernetes.io/general-worker
    value: true
evict: true #should nodes be kicked off?
daemonsets:
  - namespace: kiam
    name: kiam-agent
```

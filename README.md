# rollingpodset
Kubernetes Operator to manage and cycle Pods. Completely unnecessary. 

Acts a bit like a ReplicaSet not a Deployment, in terms of guaranteeing the right number
of Pods running at any time. 
On top of that it also deletes and recreates Pods once they have been running for longer 
than `CycleTime`, a value specified on the Custom Resource. 

Built using [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)

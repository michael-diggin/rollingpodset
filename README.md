# rollingpodset
Kubernetes Operator to manage and cycle Pods. It is completely unnecessary. 

Acts a bit like a ReplicaSet rather than a Deployment, in terms of guaranteeing the right number
of Pods running at any time (but no rolling update strategy). 
On top of that it also deletes and recreates Pods once they have been running for longer 
than `CycleTime`, a value specified on the Custom Resource. 

It's very basic, some more functioanily *may* be added at some point.

Built using [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)

## Try it out
Fairly straightforward to demo this operator and custom resource.

Having a local running cluster (eg Kind), install the CRD and Roles with
```
make install
```

Then build and deploy the Manager/Operator (this will deploy it into the namespace `rollingpodset-system`)
```
make docker-build
kind load docker-image mdiggin/controller:v1
make deploy
```

Finally deploy a sample CR
```
kubectl apply -f config/samples/apps_v1_rollingpodset.yaml 
```

Watching the pods in the default namespace will show they being terminated and recreated every `cycleTime` seconds (60 in the sample CR yaml)
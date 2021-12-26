/*
Copyright 2021.

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
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/michael-diggin/rollingpodset/api/v1"
)

var (
	podOwnerKey           = ".metadata.controller"
	apiGVStr              = appsv1.GroupVersion.String()
	startedTimeAnnotation = "apps.michael-diggin.github.io/started-at"
)

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

// RollingPodSetReconciler reconciles a RollingPodSet object
type RollingPodSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock
	RequeueTime time.Duration
}

//+kubebuilder:rbac:groups=apps.michael-diggin.github.io,resources=rollingpodsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.michael-diggin.github.io,resources=rollingpodsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.michael-diggin.github.io,resources=rollingpodsets/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RollingPodSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RollingPodSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconcile request for rps")

	var rollingPodSet appsv1.RollingPodSet
	if err := r.Get(ctx, req.NamespacedName, &rollingPodSet); err != nil {
		log.Error(err, "unable to fetch RollingPodSet")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.InNamespace(req.Namespace), client.MatchingFields{podOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child pods")
		return ctrl.Result{RequeueAfter: r.RequeueTime}, err
	}

	pods := podList.Items

	msg := fmt.Sprintf("found %d child pods", len(pods))
	log.Info(msg)

	if rollingPodSet.Status.AvailableReplicas != len(pods) {
		rollingPodSet.Status.AvailableReplicas = len(pods)
		rollingPodSet.Status.Replicas = rollingPodSet.Spec.Replicas
		if err := r.Status().Update(ctx, &rollingPodSet); err != nil {
			log.Error(err, "failed to update status")
			return ctrl.Result{RequeueAfter: r.RequeueTime}, err
		}
	}

	// sort by StartTime in descending order
	sort.Slice(pods, func(i, j int) bool {
		return pods[i].Status.StartTime.Before(pods[j].Status.StartTime)
	})

	// if num pods < replicas -> creare more
	// if num pods > replicas -> delete some
	// if now - pod start time > cycle time -> delete + recreate
	// should really be checking the status of the pods

	// loop through the list once
	// add to list for deletion if less < 0
	// add to list if too old and increment newCount

	newPodCount := 0
	less := rollingPodSet.Spec.Replicas - len(pods)
	if less > 0 {
		newPodCount = less
	}
	delPods := make([]corev1.Pod, 0, len(pods))

	now := time.Now().UTC().Unix()
	for _, p := range pods {
		if less < 0 {
			delPods = append(delPods, p)
			less++
			continue
		}
		startTime, err := getPodStartedTime(&p)
		if err != nil {
			log.Error(err, "failed to get pod started time annotation")
			return ctrl.Result{RequeueAfter: r.RequeueTime}, err
		}
		podDuration := now - startTime.Unix()
		msg := fmt.Sprintf("Pod duration %d seconds", podDuration)
		log.Info(msg)
		if podDuration > rollingPodSet.Spec.CycleTime {
			delPods = append(delPods, p)
			newPodCount++
		}
	}

	if newPodCount > 0 {
		// create the extra ones
		for i := 0; i < newPodCount; i++ {
			log.Info("creating a new pod")
			startedTime := time.Now().UTC()
			pod, err := r.constructPod(&rollingPodSet, &startedTime)
			if err != nil {
				log.Error(err, "unable to construct new pod")
				return ctrl.Result{RequeueAfter: r.RequeueTime}, err
			}
			if err := r.Create(ctx, pod); err != nil {
				log.Error(err, "unable to create pod")
				return ctrl.Result{RequeueAfter: r.RequeueTime}, err
			}
		}
	}

	if len(delPods) > 0 {
		// delete pods
		for _, p := range delPods {
			log.Info("deleting a pod")
			if err := r.Delete(ctx, &p); err != nil {
				log.Error(err, "unable to delete extra pod")
				return ctrl.Result{RequeueAfter: r.RequeueTime}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: r.RequeueTime}, nil
}

func (r *RollingPodSetReconciler) constructPod(rollPodSet *appsv1.RollingPodSet, startedTime *time.Time) (*corev1.Pod, error) {
	name := fmt.Sprintf("%s-%d", rollPodSet.Name, time.Now().UnixNano())

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   rollPodSet.Namespace,
		},
		Spec: *rollPodSet.Spec.PodTemplate.Spec.DeepCopy(),
	}
	for k, v := range rollPodSet.Spec.PodTemplate.Annotations {
		pod.Annotations[k] = v
	}
	pod.Annotations[startedTimeAnnotation] = startedTime.Format(time.RFC3339)
	for k, v := range rollPodSet.Spec.PodTemplate.Labels {
		pod.Labels[k] = v
	}
	if err := ctrl.SetControllerReference(rollPodSet, pod, r.Scheme); err != nil {
		return nil, err
	}

	return pod, nil
}

func getPodStartedTime(pod *corev1.Pod) (*time.Time, error) {
	timeRaw := pod.Annotations[startedTimeAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}
	return &timeParsed, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RollingPodSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podOwnerKey, func(rawObj client.Object) []string {
		// grab the pod object, extract the owner...
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}
		// ...make sure it's a RollingPodSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "RollingPodSet" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.RollingPodSet{}).
		//Owns(&corev1.Pod{}). usually would want this but it's spamming
		// and the cache isn't updated in time
		// could try and filter out Pod Creation events?
		Complete(r)
}

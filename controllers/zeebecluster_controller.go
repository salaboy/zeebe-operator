/*

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
	"github.com/go-logr/logr"
	guuid "github.com/google/uuid"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/test/builder"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	//"k8s.io/helm/pkg/helm"
	//"k8s.io/helm/pkg/helm/portforwarder"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	zeebev1 "zeebe-operator/api/v1"
)

// ZeebeClusterReconciler reconciles a ZeebeCluster object
type ZeebeClusterReconciler struct {
	Scheme *runtime.Scheme
	client.Client
	k8s kubernetes.Clientset
	pr  PipelineRunner
	Log logr.Logger
}

type PipelineRunner struct {
	tekton tekton.Clientset
	Log    logr.Logger
}

func (p *PipelineRunner) initPipelineRunner(namespace string) {
	log := p.Log.WithValues("zeebecluster", namespace)
	//@TODO:  Do as initialization of the Operator ..
	pipelineResource := builder.PipelineResource("zeebe-base-chart", namespace,
		builder.PipelineResourceSpec(v1alpha1.PipelineResourceType("git"),
			builder.PipelineResourceSpecParam("revision", "master"),
			builder.PipelineResourceSpecParam("url", "https://github.com/salaboy/zeebe-base-chart")))

	log.Info("> Creating PipelineResource: ", "pipelineResource", pipelineResource)
	p.tekton.TektonV1alpha1().PipelineResources(namespace).Create(pipelineResource)
	//@TODO: END
}

func (p *PipelineRunner) createTaskAndTaskRun(taskId string, namespace string, zeebeCluster zeebev1.ZeebeCluster, r ZeebeClusterReconciler) {
	log := p.Log.WithValues("zeebecluster", namespace)
	task := builder.Task("install-task-"+zeebeCluster.Name+"-"+taskId, zeebeCluster.Namespace,
		builder.TaskSpec(
			builder.TaskInputs(builder.InputsResource("zeebe-base-chart", "git")),
			builder.Step("clone-base-helm-chart", "gcr.io/jenkinsxio/builder-go:2.0.1028-359",
				builder.StepCommand("make", "-C", "/workspace/zeebe-base-chart/", "build", "install"),
				builder.StepEnvVar("CLUSTER_NAME", zeebeCluster.Name),
				builder.StepEnvVar("NAMESPACE", zeebeCluster.Spec.TargetNamespace))))

	if err := ctrl.SetControllerReference(&zeebeCluster, task, r.Scheme); err != nil {
		log.Error(err, "unable set owner to task")
	}

	_, errorTask := p.tekton.TektonV1alpha1().Tasks(zeebeCluster.Spec.TargetNamespace).Create(task)
	if errorTask != nil {
		log.Error(errorTask, "Erorr Creating task")
	}

	log.Info("> Creating Task: ", "task", task)

	taskRun := builder.TaskRun("install-task-run-"+zeebeCluster.Name+"-"+taskId, zeebeCluster.Namespace,
		builder.TaskRunSpec(
			builder.TaskRunServiceAccountName("pipelinerunner"),
			builder.TaskRunDeprecatedServiceAccount("pipelinerunner", "pipelinerunner"), // This require a SA being created for it to run

			builder.TaskRunTaskRef("install-task-"+zeebeCluster.Name+"-"+taskId),
			builder.TaskRunInputs(builder.TaskRunInputsResource("zeebe-base-chart",
				builder.TaskResourceBindingRef("zeebe-base-chart")))))

	if err := ctrl.SetControllerReference(&zeebeCluster, taskRun, r.Scheme); err != nil {
		log.Error(err, "unable set owner to taskRun")
	}
	log.Info("> Creating TaskRun: ", "taskrun", taskRun)
	_, errorTaskRun := p.tekton.TektonV1alpha1().TaskRuns(zeebeCluster.Spec.TargetNamespace).Create(taskRun)

	if errorTaskRun != nil {
		log.Error(errorTaskRun, "Error Creating taskRun")
	}

}

//func (r *ZeebeClusterReconciler) watchClusterResources(zeebeCluster zeebev1.ZeebeCluster, clusterName string) (status string, err error) {
//	//clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
//	ctx := context.Background()
//	api := r.k8s.AppsV1()
//	listOptions := metav1.ListOptions{
//		LabelSelector: "app.kubernetes.io/instance=" + "zeebecluster-salaboy",
//	}
//	watcher, err := api.StatefulSets("default").Watch(listOptions)
//	if err != nil {
//		r.Log.Error(err, "unable to create watch ")
//	}
//
//	ch := watcher.ResultChan()
//	// a channel to tell it to stop
//	stopchan := make(chan struct{})
//	// a channel to signal that it's stopped
//	//stoppedchan := make(chan struct{})
//
//	for {
//		select {
//		case event, ok := <-ch:
//			if !ok {
//				// the channel got closed, so we need to restart
//				r.Log.Info("Kubernetes hung up on us, restarting event watcher")
//				return "restart", nil
//			}
//
//			statefulSet, ok := event.Object.(*appsV1.StatefulSet)
//			if !ok {
//				r.Log.Info("ERROR: unexpected type")
//			}
//			switch event.Type {
//			case watch.Added:
//				r.Log.Info("StatefulSet Added: " +
//					statefulSet.Name)
//
//				_, err := ctrl.CreateOrUpdate(ctx, r.Client, statefulSet, func() error {
//					return ctrl.SetControllerReference(&zeebeCluster, statefulSet, r.Scheme)
//				})
//				if err != nil {
//					r.Log.Error(err, "Error setting up owner for statefulset")
//				}
//			case watch.Deleted:
//				r.Log.Info("Pod Deleted: " +
//					statefulSet.Name)
//			}
//
//		case <-time.After(30 * time.Minute):
//			// deal with the issue where we get no events
//			r.Log.Info("Timeout, restarting event watcher")
//			return "timeout", nil
//		case <-stopchan:
//			// stop
//			r.Log.Info("Stopping watcher")
//			return "stopped", nil
//		}
//
//	}
//
//}
//
//func (r *ZeebeClusterReconciler) createMonitor(zeebeCluster zeebev1.ZeebeCluster, clusterName string) {
//	for {
//		if status, err := r.watchClusterResources(zeebeCluster, clusterName); err != nil {
//			if status == "stopped" {
//				return
//			}
//			r.Log.Error(err, "error watching cluster: "+clusterName+" resources.")
//		}
//		time.Sleep(5 * time.Second)
//	}
//
//}

/* Reconcile should do:
	1) get CRD Cluster
    2) run pipeline to install/update
    3) update CRD status based on pods
    4) update URL
*/

// +kubebuilder:rbac:groups=zeebe.zeebe.io,resources=zeebeclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zeebe.zeebe.io,resources=zeebeclusters/status,verbs=get;update;patch

// CRUD core: namespaces, events, secrets, services and configmaps
// +kubebuilder:rbac:groups=core,resources=services;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status;configmaps/status,verbs=get

// LIST core: endpoints
// +kubebuilder:rbac:groups=core,resources=endpoints;pods,verbs=list;watch

// CRUD apps: deployments and statefulsets
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status;deployments/status,verbs=get

func (r *ZeebeClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("zeebecluster", req.NamespacedName)
	// your logic here
	var zeebeCluster zeebev1.ZeebeCluster
	if err := r.Get(ctx, req.NamespacedName, &zeebeCluster); err != nil {
		// it might be not found if this is a delete request
		if ignoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch cluster")
		return ctrl.Result{}, err
	}

	if zeebeCluster.Status.ClusterName != "" {
		return ctrl.Result{}, nil
	}

	// process the request, make some changes to the cluster, // set some status on `zeebeCluster`, etc
	// update status, since we probably changed it above
	log.Info("> Zeebe Cluster: ", "cluster", zeebeCluster)

	// Create a new Monitor here to watch for StatefulSet and update status when the resources are created
	// The main problem here is that helm will install resources.. and here I need to search for those based on labels/names

	r.pr.initPipelineRunner("default")

	var taskId = guuid.New().String()[0:8]
	var clusterName = zeebeCluster.Name + "-" + taskId

	r.pr.createTaskAndTaskRun(taskId, "default", zeebeCluster, *r)

	// Create watch inside goroutine to look for resources matching the name of the cluster (labels) and set ownerreference
	// Also monitor for status

	zeebeCluster.Status.ClusterName = clusterName

	log.Info("> Zeebe Cluster Name: " + clusterName)

	//r.createMonitor(zeebeCluster, clusterName)

	if err := r.Status().Update(ctx, &zeebeCluster); err != nil {
		log.Error(err, "unable to update cluster spec")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ZeebeClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// create the clientset
	clientSet, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		panic(err.Error())
	}
	r.k8s = *clientSet

//	//port forward tiller
//	tillerTunnel, _ := portforwarder.New("kube-system", clientSet, mgr.GetConfig())
//
////	new helm client
//	helmClient := helm.NewClient()
//
//	//list/print releases
//	resp, _ := helmClient.ListReleases()
//	for _, release := range resp.Releases {
//		r.Log.Info(">> Release Name" + release.GetName())
//	}

	tektonClientSet, _ := tekton.NewForConfig(mgr.GetConfig())
	r.pr.tekton = *tektonClientSet
	r.pr.Log = r.Log
	r.Scheme = mgr.GetScheme()
	return ctrl.NewControllerManagedBy(mgr).
		For(&zeebev1.ZeebeCluster{}).

		//Owns(&core.ConfigMap{}).
		Owns(&coreV1.Service{}).
		Owns(&appsV1.StatefulSet{}).
		Watches(&source.Kind{Type: &appsV1.StatefulSet{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []ctrl.Request {
				statefulSet, ok := obj.Object.(*appsV1.StatefulSet)
				if !ok {
					r.Log.Info("ERROR: unexpected type")
				}

				r.Log.Info("StatefulSet watch: " + statefulSet.Name)
				r.Log.Info("StatefulSet Labels: ", "labels", statefulSet.GetLabels())
				r.Log.Info("StatefulSet Labels: ", "appLabel", statefulSet.GetLabels()["app.kubernetes.io/instance"])

				var zeebeClusterList zeebev1.ZeebeClusterList
				//client.MatchingLabels{"name" :statefulSet.GetLabels()["app.kubernetes.io/instance"] }
				if err := r.List(context.Background(), &zeebeClusterList); err != nil {
					r.Log.Info("unable to get zeebe clusters for statefulset", "statefulset", obj.Meta.GetName())
					return nil
				}
				r.Log.Info("List: ", "list", zeebeClusterList.Items)
				r.Log.Info("Length: ", "length", len(zeebeClusterList.Items))
				if len(zeebeClusterList.Items) == 1 {
					if zeebeClusterList.Items[0].Name == statefulSet.GetLabels()["app.kubernetes.io/instance"] {
						if zeebeClusterList.Items[0].OwnerReferences == nil {
							_, err := ctrl.CreateOrUpdate(context.Background(), r.Client, statefulSet, func() error {
								r.Log.Info("Zeebe Cluster found, updating statefulset ownership ", "cluster", zeebeClusterList.Items[0].Name)
								return ctrl.SetControllerReference(&zeebeClusterList.Items[0], statefulSet, r.Scheme)
							})
							if err != nil {
								r.Log.Error(err, "Error setting up owner for statefulset")
							}
						}
					}
				} else {
					r.Log.Info(">> No Zeebe Cluster matching label: " + statefulSet.GetLabels()["app.kubernetes.io/instance"])
				}

				return nil

			}),
		}).
		Watches(&source.Kind{Type: &coreV1.Service{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []ctrl.Request {
				service, ok := obj.Object.(*coreV1.Service)
				if !ok {
					r.Log.Info("ERROR: unexpected type")
				}

				r.Log.Info("Service watch: " + service.Name)
				r.Log.Info("Service Labels All: ", "labels", service.GetLabels())
				r.Log.Info("Service App Label: ", "appLabel", service.GetLabels()["app.kubernetes.io/instance"])

				var zeebeClusterList zeebev1.ZeebeClusterList
				//, client.MatchingFields{"metadata.name" :service.GetLabels()["app.kubernetes.io/instance"]}
				if err := r.List(context.Background(), &zeebeClusterList); err != nil {
					r.Log.Info("unable to get zeebe clusters for statefulset", "statefulset", obj.Meta.GetName())
					return nil
				}
				r.Log.Info("List: ", "list", zeebeClusterList.Items)
				r.Log.Info("Length: ", "length", len(zeebeClusterList.Items))
				if len(zeebeClusterList.Items) == 1 {
					if zeebeClusterList.Items[0].Name == service.GetLabels()["app.kubernetes.io/instance"] {
						if zeebeClusterList.Items[0].OwnerReferences == nil {
							_, err := ctrl.CreateOrUpdate(context.Background(), r.Client, service, func() error {
								r.Log.Info("Zeebe Cluster found, updating service ownership ", "cluster", zeebeClusterList.Items[0].Name)
								return ctrl.SetControllerReference(&zeebeClusterList.Items[0], service, r.Scheme)
							})
							if err != nil {
								r.Log.Error(err, "Error setting up owner for service")
							}
						}
					}
				} else {
					r.Log.Info(">> No Zeebe Cluster matching label: " + service.GetLabels()["app.kubernetes.io/instance"])
				}

				return nil

			}),
		}).
		Watches(&source.Kind{Type: &coreV1.ConfigMap{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []ctrl.Request {
				service, ok := obj.Object.(*coreV1.ConfigMap)
				if !ok {
					r.Log.Info("ERROR: unexpected type")
				}

				r.Log.Info("ConfigMap watch: " + service.Name)

				var zeebeClusterList zeebev1.ZeebeClusterList
				if err := r.List(context.Background(), &zeebeClusterList); err != nil {
					r.Log.Info("unable to get zeebe clusters for configMap", "configMap", obj.Meta.GetName())
					return nil
				}

				if len(zeebeClusterList.Items) == 1 {
					if zeebeClusterList.Items[0].Name == service.GetLabels()["app.kubernetes.io/instance"] {
						if zeebeClusterList.Items[0].OwnerReferences == nil {
							_, err := ctrl.CreateOrUpdate(context.Background(), r.Client, service, func() error {
								r.Log.Info("Zeebe Cluster found, updating ConfigMap ownership ", "cluster", zeebeClusterList.Items[0].Name)
								return ctrl.SetControllerReference(&zeebeClusterList.Items[0], service, r.Scheme)
							})
							if err != nil {
								r.Log.Error(err, "Error setting up owner for configMap")
							}
						}
					}
				} else {
					r.Log.Info(">> No Zeebe Cluster matching label: " + service.GetLabels()["app.kubernetes.io/instance"])
				}

				return nil

			}),
		}).
		Complete(r)
}

func ignoreNotFound(err error) error {
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func ownedByOther(obj metav1.Object, apiVersion schema.GroupVersion, kind, name string) *metav1.OwnerReference {
	if ownerRef := metav1.GetControllerOf(obj); ownerRef != nil && (ownerRef.Name != name || ownerRef.Kind != kind || ownerRef.APIVersion != apiVersion.String()) {
		return ownerRef
	}
	return nil
}

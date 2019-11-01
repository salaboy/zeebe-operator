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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	zeebev1 "zeebe-operator/api/v1"

	guuid "github.com/google/uuid"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/test/builder"
)

// ZeebeClusterReconciler reconciles a ZeebeCluster object
type ZeebeClusterReconciler struct {
	client.Client
	tekton tekton.Clientset
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=zeebe.zeebe.io,resources=zeebeclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zeebe.zeebe.io,resources=zeebeclusters/status,verbs=get;update;patch

func (r *ZeebeClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("zeebecluster", req.NamespacedName)
	// your logic here
	var app zeebev1.ZeebeCluster
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		// it might be not found if this is a delete request
		//if ignoreNotFound(err) == nil { return ctrl.Result{}, nil
		//}
		log.Error(err, "unable to fetch cluster")
		return ctrl.Result{}, err
	}
	// process the request, make some changes to the cluster, // set some status on `app`, etc
	// update status, since we probably changed it above
	log.Info("> Zeebe Cluster: ", "cluster", app)

	//if err := r.Update(ctx, &app); err != nil {
	//	log.Error(err, "unable to update cluster spec")
	//	return ctrl.Result{}, err
	//}

	//@Questions:
	// 1) should I create tekton resources in default namespace?
	// 2) should I craete one task + task run per Zeebe Cluster? I think that it make sense

	//@TODO:  Do as initialization of the Operator ..
	pipelineResource := builder.PipelineResource("zeebe-base-chart", app.Namespace,
		builder.PipelineResourceSpec(v1alpha1.PipelineResourceType("git"),
			builder.PipelineResourceSpecParam("revision", "master"),
			builder.PipelineResourceSpecParam("url", "https://github.com/salaboy/zeebe-base-chart")))

	log.Info("> Creating PipelineResource: ", "pipelineResource", pipelineResource)
	r.tekton.TektonV1alpha1().PipelineResources(app.Namespace).Create(pipelineResource)
	//@TODO: END

	// I should create a task per ZeebeCluster ... Here I need to make this task generation random as it has also parameters that might change.
	task := builder.Task("zeebe-install-task-"+app.Name, app.Namespace,
		builder.TaskSpec(
			builder.TaskInputs(builder.InputsResource("zeebe-base-chart", "git")),
			builder.Step("clone-base-helm-chart", "gcr.io/jenkinsxio/builder-go",
				builder.StepCommand("make", "-C", "/workspace/zeebe-base-chart/", "build", "install"),
				builder.StepEnvVar("CLUSTER_NAME", app.Name),
				builder.StepEnvVar("NAMESPACE", app.Spec.TargetNamespace))))

	r.tekton.TektonV1alpha1().Tasks(app.Namespace).Create(task)

	taskRun := builder.TaskRun("zeebe-install-task-run-"+guuid.New().String(), app.Namespace,
		builder.TaskRunSpec(
			builder.TaskRunServiceAccountName("pipelinerunner"),
			builder.TaskRunDeprecatedServiceAccount("pipelinerunner", "pipelinerunner"), // This require a SA being created for it to run

			builder.TaskRunTaskRef("zeebe-install-task-"+app.Name),
			builder.TaskRunInputs(builder.TaskRunInputsResource("zeebe-base-chart",
				builder.TaskResourceBindingRef("zeebe-base-chart")))))

	r.tekton.TektonV1alpha1().TaskRuns(app.Namespace).Create(taskRun)

	return ctrl.Result{}, nil
}

func (r *ZeebeClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tektonClientSet, _ := tekton.NewForConfig(mgr.GetConfig())
	r.tekton = *tektonClientSet

	return ctrl.NewControllerManagedBy(mgr).
		For(&zeebev1.ZeebeCluster{}).
		Complete(r)
}

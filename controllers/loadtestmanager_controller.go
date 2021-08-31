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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	macav1alpha1 "github.com/itohdak/maca-operator/api/v1alpha1"
)

// LoadTestManagerReconciler reconciles a LoadTestManager object
type LoadTestManagerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=maca.itohdak.github.com,resources=loadtestmanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=maca.itohdak.github.com,resources=loadtestmanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=maca.itohdak.github.com,resources=loadtestmanagers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LoadTestManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *LoadTestManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ltManager macav1alpha1.LoadTestManager
	err := r.Get(ctx, req.NamespacedName, &ltManager)
	if errors.IsNotFound(err) {
		// r.removeMetrics(ltManager)
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "failed to get LoadTestManager", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	if err = r.reconcileConfigMap(ctx, ltManager); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.reconcileConfigMap2(ctx, ltManager); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.reconcileDeployment(ctx, ltManager); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.reconcileDeployment2(ctx, ltManager); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.reconcileService(ctx, ltManager); err != nil {
		return ctrl.Result{}, err
	}

	return r.updateStatus(ctx, ltManager)
}

func (r *LoadTestManagerReconciler) reconcileConfigMap(ctx context.Context, ltManager macav1alpha1.LoadTestManager) error {
	logger := log.FromContext(ctx)

	cm := &corev1.ConfigMap{}
	cm.SetNamespace(ltManager.Namespace)
	cm.SetName("locust-scripts-" + ltManager.Name)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		for name, content := range ltManager.Spec.Scripts {
			cm.Data[name] = content
		}
		return ctrl.SetControllerReference(&ltManager, cm, r.Scheme)
	})

	if err != nil {
		logger.Error(err, "failed to create or update ConfigMap")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconciled ConfigMap successfully", "op", op)
	}
	return nil
}

func (r *LoadTestManagerReconciler) reconcileConfigMap2(ctx context.Context, ltManager macav1alpha1.LoadTestManager) error {
	logger := log.FromContext(ctx)

	cm := &corev1.ConfigMap{}
	cm.SetNamespace(ltManager.Namespace)
	cm.SetName("locust-cm-" + ltManager.Name)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		for name, content := range ltManager.Spec.TargetHost {
			cm.Data[name] = content
		}
		return ctrl.SetControllerReference(&ltManager, cm, r.Scheme)
	})

	if err != nil {
		logger.Error(err, "failed to create or update ConfigMap")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconciled ConfigMap successfully", "op", op)
	}
	return nil
}

func (r *LoadTestManagerReconciler) reconcileDeployment(ctx context.Context, ltManager macav1alpha1.LoadTestManager) error {
	logger := log.FromContext(ctx)

	depName := "locust-master-" + ltManager.Name

	owner, err := ownerRef(ltManager, r.Scheme)
	if err != nil {
		return err
	}

	labels := map[string]string{
		"role": "locust-master",
	}
	dep := appsv1apply.Deployment(depName, ltManager.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(1). // only one replica for master
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(labels)).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(labels).
				WithSpec(corev1apply.PodSpec().
					WithContainers(corev1apply.Container().
						WithName("locust-master").
						WithImage("grubykarol/locust:1.2.3-python3.9-alpine3.12").
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithEnv(
							corev1apply.EnvVar().
								WithName("ATTACKED_HOST").
								WithValueFrom(corev1apply.EnvVarSource().
									WithConfigMapKeyRef(corev1apply.ConfigMapKeySelector().
										WithKey("ATTACKED_HOST").
										WithName("locust-cm-"+ltManager.Name),
									),
								),
							corev1apply.EnvVar().
								WithName("LOCUST_MODE").
								WithValue("master"),
						).
						WithVolumeMounts(corev1apply.VolumeMount().
							WithName("locust-scripts").
							WithMountPath("/locust"),
						).
						WithPorts(
							corev1apply.ContainerPort().
								WithName("web-ui").
								WithProtocol(corev1.ProtocolTCP).
								WithContainerPort(8089),
							corev1apply.ContainerPort().
								WithName("comm").
								WithProtocol(corev1.ProtocolTCP).
								WithContainerPort(5557),
							corev1apply.ContainerPort().
								WithName("comm-plus-1").
								WithProtocol(corev1.ProtocolTCP).
								WithContainerPort(5558),
						),
					).
					WithNodeSelector(map[string]string{
						"app": "locust",
					}).
					WithTolerations(corev1apply.Toleration().
						WithKey("run").
						WithOperator("Equal").
						WithValue("locust").
						WithEffect("NoSchedule"),
					).
					WithVolumes(corev1apply.Volume().
						WithName("locust-scripts").
						WithConfigMap(corev1apply.ConfigMapVolumeSource().
							WithName("locust-scripts-" + ltManager.Name),
						),
					),
				),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: ltManager.Namespace, Name: depName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := appsv1apply.ExtractDeployment(&current, "load-test-manager-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(dep, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "load-test-manager-controller",
		Force:        pointer.Bool(true),
	})

	if err != nil {
		logger.Error(err, "failed to create or update Deployment")
		return err
	}
	logger.Info("reconciled Deployment successfully", "name", ltManager.Name)
	return nil
}

func (r *LoadTestManagerReconciler) reconcileDeployment2(ctx context.Context, ltManager macav1alpha1.LoadTestManager) error {
	logger := log.FromContext(ctx)

	depName := "locust-slave-" + ltManager.Name

	owner, err := ownerRef(ltManager, r.Scheme)
	if err != nil {
		return err
	}

	labels := map[string]string{
		"role": "locust-slave",
	}
	dep := appsv1apply.Deployment(depName, ltManager.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(ltManager.Spec.Replicas).
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(labels)).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(labels).
				WithSpec(corev1apply.PodSpec().
					WithContainers(corev1apply.Container().
						WithName("locust-slave").
						WithImage("grubykarol/locust:1.2.3-python3.9-alpine3.12").
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithEnv(
							corev1apply.EnvVar().
								WithName("ATTACKED_HOST").
								WithValueFrom(corev1apply.EnvVarSource().
									WithConfigMapKeyRef(corev1apply.ConfigMapKeySelector().
										WithKey("ATTACKED_HOST").
										WithName("locust-cm-"+ltManager.Name),
									),
								),
							corev1apply.EnvVar().
								WithName("LOCUST_MODE").
								WithValue("worker"),
							corev1apply.EnvVar().
								WithName("LOCUST_MASTER_HOST").
								WithValue("locust-master-"+ltManager.Name),
						).
						WithVolumeMounts(corev1apply.VolumeMount().
							WithName("locust-scripts").
							WithMountPath("/locust"),
						),
					).
					WithNodeSelector(map[string]string{
						"app": "locust",
					}).
					WithTolerations(corev1apply.Toleration().
						WithKey("run").
						WithOperator("Equal").
						WithValue("locust").
						WithEffect("NoSchedule"),
					).
					WithVolumes(corev1apply.Volume().
						WithName("locust-scripts").
						WithConfigMap(corev1apply.ConfigMapVolumeSource().
							WithName("locust-scripts-" + ltManager.Name),
						),
					),
				),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: ltManager.Namespace, Name: depName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := appsv1apply.ExtractDeployment(&current, "load-test-manager-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(dep, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "load-test-manager-controller",
		Force:        pointer.Bool(true),
	})

	if err != nil {
		logger.Error(err, "failed to create or update Deployment")
		return err
	}
	logger.Info("reconciled Deployment successfully", "name", ltManager.Name)
	return nil
}

func (r *LoadTestManagerReconciler) reconcileService(ctx context.Context, ltManager macav1alpha1.LoadTestManager) error {
	logger := log.FromContext(ctx)
	// svcName := "load-test-service-" + ltManager.Name
	svcName := "locust-master-" + ltManager.Name

	owner, err := ownerRef(ltManager, r.Scheme)
	if err != nil {
		return err
	}

	labels := map[string]string{
		"role": "locust-master",
	}
	svc := corev1apply.Service(svcName, ltManager.Namespace).
		WithLabels(labels).
		WithOwnerReferences(owner).
		WithSpec(corev1apply.ServiceSpec().
			WithSelector(labels).
			WithType(corev1.ServiceTypeClusterIP).
			WithPorts(
				corev1apply.ServicePort().
					WithName("web-ui").
					WithProtocol(corev1.ProtocolTCP).
					WithPort(8089).
					WithTargetPort(intstr.FromInt(8089)),
				corev1apply.ServicePort().
					WithName("communication").
					WithProtocol(corev1.ProtocolTCP).
					WithPort(5557).
					WithTargetPort(intstr.FromInt(5557)),
				corev1apply.ServicePort().
					WithName("communication-plus-1").
					WithProtocol(corev1.ProtocolTCP).
					WithPort(5558).
					WithTargetPort(intstr.FromInt(5558)),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(svc)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current corev1.Service
	err = r.Get(ctx, client.ObjectKey{Namespace: ltManager.Namespace, Name: svcName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := corev1apply.ExtractService(&current, "load-test-manager-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(svc, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "load-test-manager-controller",
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "failed to create or update Service")
		return err
	}

	logger.Info("reconciled Service successfully", "name", ltManager.Name)
	return nil
}

func ownerRef(ltManager macav1alpha1.LoadTestManager, scheme *runtime.Scheme) (*metav1apply.OwnerReferenceApplyConfiguration, error) {
	gvk, err := apiutil.GVKForObject(&ltManager, scheme)
	if err != nil {
		return nil, err
	}
	ref := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(ltManager.Name).
		WithUID(ltManager.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)
	return ref, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LoadTestManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&macav1alpha1.LoadTestManager{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *LoadTestManagerReconciler) updateStatus(ctx context.Context, ltManager macav1alpha1.LoadTestManager) (ctrl.Result, error) {
	var dep appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{Namespace: ltManager.Namespace, Name: "locust-slave-" + ltManager.Name}, &dep)
	if err != nil {
		return ctrl.Result{}, err
	}

	var status macav1alpha1.LoadTestManagerStatus
	if dep.Status.AvailableReplicas == 0 {
		status = macav1alpha1.LoadTestManagerNotReady
	} else if dep.Status.AvailableReplicas == ltManager.Spec.Replicas {
		status = macav1alpha1.LoadTestManagerAvailable
	} else { // TODO
		status = macav1alpha1.LoadTestManagerTesting
	}

	if ltManager.Status != status {
		ltManager.Status = status
		// r.setMetrics(ltManager)

		// r.Recorder.Event(&ltManager, corev1.EventTypeNormal, "Updated", fmt.Sprintf("LoadTestManager(%s:%s) updated: %s", ltManager.Namespace, ltManager.Name, ltManager.Status))

		err = r.Status().Update(ctx, &ltManager)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if ltManager.Status == macav1alpha1.LoadTestManagerNotReady {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// func (r *LoadTestManagerReconciler) setMetrics(ltManager macav1alpha1.LoadTestManager) {
// 	switch ltManager.Status {
// 	case macav1alpha1.LoadTestManagerNotReady:
// 		metrics.NotReadyVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(1)
// 		metrics.AvailableVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(0)
// 		metrics.HealthyVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(0)
// 	case macav1alpha1.LoadTestManagerAvailable:
// 		metrics.NotReadyVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(0)
// 		metrics.AvailableVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(1)
// 		metrics.HealthyVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(0)
// 	case macav1alpha1.LoadTestManagerTesting:
// 		metrics.NotReadyVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(0)
// 		metrics.AvailableVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(0)
// 		metrics.HealthyVec.WithLabelValues(ltManager.Name, ltManager.Name).Set(1)
// 	}
// }

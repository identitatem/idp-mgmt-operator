// Copyright Red Hat

package installer

import (
	"context"
	"fmt"
	"os"
	"strings"

	// "fmt"
	// "os"

	giterrors "github.com/pkg/errors"

	"github.com/ghodss/yaml"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	// "sigs.k8s.io/controller-runtime/pkg/handler"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	// "sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpoperatorconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/deploy"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	//+kubebuilder:scaffold:imports
)

// IDPConfigReconciler reconciles a Strategy object
type IDPConfigReconciler struct {
	client.Client
	KubeClient         kubernetes.Interface
	DynamicClient      dynamic.Interface
	APIExtensionClient apiextensionsclient.Interface
	Log                logr.Logger
	Scheme             *runtime.Scheme
}

var podName, podNamespace string

// +kubebuilder:rbac:groups="",resources={namespaces, pods},verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources={services,serviceaccounts},verbs=get;create;update;list;watch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={roles,rolebindings,clusterrolebindings},verbs=get;create;update;list;watch;delete
// +kubebuilder:rbac:groups="apps",resources={deployments},verbs=get;create;update;list;watch;delete

// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources={validatingwebhookconfigurations},verbs=get;create;update;list;watch;delete
// +kubebuilder:rbac:groups="apiregistration.k8s.io",resources={apiservices},verbs=get;create;update;list;watch;delete

// +kubebuilder:rbac:groups="identityconfig.identitatem.io",resources={idpconfigs},verbs=get;create;update;list;watch;delete

// +kubebuilder:rbac:groups="multicluster.openshift.io",resources={multiclusterengines},verbs=get;list;watch
// +kubebuilder:rbac:groups="operator.open-cluster-management.io",resources={multiclusterhubs},verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Strategy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *IDPConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("namespace", req.NamespacedName, "name", req.Name)

	// your logic here
	instance := &identitatemv1alpha1.IDPConfig{}

	if err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: req.Namespace, Name: req.Name},
		instance,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	r.Log.Info("Instance", "instance", instance)
	r.Log.Info("Running Reconcile for IDP Config", "Name: ", instance.GetName(), " Namespace:", instance.GetNamespace())

	if instance.DeletionTimestamp != nil {
		if err := r.processIDPConfigDeletion(instance); err != nil {
			return reconcile.Result{}, err
		}
		r.Log.Info("remove finalizer", "Finalizer:", helpers.AuthrealmFinalizer, "name", instance.Name, "namespace", instance.Namespace)
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer on idpconfig to make sure the installer process it.
	controllerutil.AddFinalizer(instance, helpers.AuthrealmFinalizer)

	if err := r.Client.Update(context.TODO(), instance); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	if err := r.processIDPConfigCreation(instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IDPConfigReconciler) processIDPConfigCreation(idpConfig *identitatemv1alpha1.IDPConfig) error {
	r.Log.Info("processIDPConfigCreation", "Name", idpConfig.Name, "Namespace", idpConfig.Namespace)
	pod := &corev1.Pod{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: podNamespace}, pod); err != nil {
		return err
	}
	r.Log.Info("Pod", "Name", pod.Name, "Namespace", pod.Namespace, "deletiontimeStamp", pod.DeletionTimestamp)
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()
	readerDeploy := deploy.GetScenarioResourcesReader()

	//Deploy dex operator
	files := []string{
		"idp-mgmt-operator/service_account.yaml",
		"idp-mgmt-operator/leader_election_role.yaml",
		"idp-mgmt-operator/leader_election_role_binding.yaml",
		"idp-mgmt-operator/clusterrole.yaml",
		"idp-mgmt-operator/clusterrole_binding.yaml",
	}

	image := pod.Spec.Containers[0].Image
	values := struct {
		Image            string
		Namespace        string
		ImageDexOperator string
		ImageDexServer   string
	}{
		Image:            image,
		Namespace:        podNamespace,
		ImageDexOperator: os.Getenv("RELATED_IMAGE_DEX_OPERATOR"),
		ImageDexServer:   os.Getenv("RELATED_IMAGE_DEX_SERVER"),
	}

	_, err := applier.ApplyDirectly(readerDeploy, values, false, "", files...)
	if err != nil {
		return giterrors.WithStack(err)
	}

	files = []string{
		"idp-mgmt-operator/manager.yaml",
	}

	if strings.Contains(pod.Spec.Containers[0].Image, "coverage") {
		files = []string{
			"idp-mgmt-operator/manager_coverage.yaml",
		}
	}

	_, err = applier.ApplyDeployments(readerDeploy, values, false, "", files...)
	if err != nil {
		return giterrors.WithStack(err)
	}

	// Do not deploy webhook on functional test as they run on KinD
	// and so does'nt have the openshift secret cert generation
	// TODO generate a secret cert in the test suite.
	if strings.Contains(pod.Spec.Containers[0].Image, "coverage") {
		return nil
	}
	//Deploy webhook
	files = []string{
		"webhook/service_account.yaml",
		"webhook/webhook_clusterrole.yaml",
		"webhook/webhook_clusterrolebinding.yaml",
		"webhook/webhook_service.yaml",
	}

	_, err = applier.ApplyDirectly(readerDeploy, values, false, "", files...)
	if err != nil {
		return giterrors.WithStack(err)
	}

	files = []string{
		"webhook/webhook.yaml",
	}

	_, err = applier.ApplyDeployments(readerDeploy, values, false, "", files...)
	if err != nil {
		return giterrors.WithStack(err)
	}

	b, err := applier.MustTempalteAsset(readerDeploy, values, "", "webhook/webhook_validating_config.yaml")
	if err != nil {
		return giterrors.WithStack(err)
	}
	validationWebhookConfiguration := &admissionregistration.ValidatingWebhookConfiguration{}
	err = yaml.Unmarshal(b, validationWebhookConfiguration)
	if err != nil {
		return giterrors.WithStack(err)
	}
	if err := r.Client.Create(context.TODO(), validationWebhookConfiguration, &client.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return giterrors.WithStack(err)
		}
	}

	b, err = applier.MustTempalteAsset(readerDeploy, values, "", "webhook/webhook_apiservice.yaml")
	if err != nil {
		return giterrors.WithStack(err)
	}
	if err != nil {
		return giterrors.WithStack(err)
	}
	apiService := &apiregistrationv1.APIService{}
	err = yaml.Unmarshal(b, apiService)
	if err != nil {
		return giterrors.WithStack(err)
	}
	if err := r.Client.Create(context.TODO(), apiService, &client.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return giterrors.WithStack(err)
		}
	}

	return nil
}

func (r *IDPConfigReconciler) processIDPConfigDeletion(idpConfig *identitatemv1alpha1.IDPConfig) error {
	r.Log.Info("processIDPConfigDeletion", "Name", idpConfig.Name, "Namespace", idpConfig.Namespace)
	//Delete operator deployment
	r.Log.Info("Delete deployment", "name", "idp-mgmt-operator-manager", "namespace", podNamespace)
	idpMgmtOperatorDeployment := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-operator-manager", Namespace: podNamespace}, idpMgmtOperatorDeployment)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), idpMgmtOperatorDeployment, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete roleBinding", "name", "idp-mgmt-operator-leader-election-rolebinding", "namespace", podNamespace)
	idpMgmtOperatorLeaderElectionRoleBinding := &rbacv1.RoleBinding{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-operator-leader-election-rolebinding", Namespace: podNamespace}, idpMgmtOperatorLeaderElectionRoleBinding)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), idpMgmtOperatorLeaderElectionRoleBinding, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete ClusterRoleBinding", "name", "idp-mgmt-operator-manager-rolebinding", "namespace", podNamespace)
	idpMgmtOperatorClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-operator-manager-rolebinding", Namespace: podNamespace}, idpMgmtOperatorClusterRoleBinding)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), idpMgmtOperatorClusterRoleBinding, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete serviceAccount", "name", "idp-mgmt-operator-manager", "namespace", podNamespace)
	idpMgmtOperatorServiceAccount := &corev1.ServiceAccount{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-operator-manager", Namespace: podNamespace}, idpMgmtOperatorServiceAccount)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), idpMgmtOperatorServiceAccount, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete ClusterRole", "name", "idp-mgmt-operator-manager-role", "namespace", podNamespace)
	idpMgmtOperatorClusterRole := &rbacv1.ClusterRole{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-operator-manager-role"}, idpMgmtOperatorClusterRole)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), idpMgmtOperatorClusterRole, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete Role", "name", "leader-election-operator-role", "namespace", podNamespace)
	idpMgmtOperatorRole := &rbacv1.Role{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "leader-election-operator-role", Namespace: podNamespace}, idpMgmtOperatorRole)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), idpMgmtOperatorRole, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	// Do not delete webhook on functional test as it is not installed
	pod := &corev1.Pod{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: podNamespace}, pod); err != nil {
		return err
	}
	r.Log.Info("Pod", "Name", pod.Name, "Namespace", pod.Namespace, "deletiontimeStamp", pod.DeletionTimestamp)
	if strings.Contains(pod.Spec.Containers[0].Image, "coverage") {
		return nil
	}

	//Delete webhook
	r.Log.Info("Delete Deployment", "name", "idp-mgmt-webhook-service", "namespace", podNamespace)
	webhookDeployment := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-webhook-service", Namespace: podNamespace}, webhookDeployment)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), webhookDeployment, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete APIService", "name", "v1alpha1.admission.identityconfig.identitatem.io")
	apiService := &apiregistrationv1.APIService{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "v1alpha1.admission.identityconfig.identitatem.io"}, apiService)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), apiService, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete ClusterRoleBinding", "name", "idp-mgmt-webhook-service")
	webHookClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-webhook-service"}, webHookClusterRoleBinding)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), webHookClusterRoleBinding, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete ClusterRole", "name", "idp-mgmt-webhook-service")
	webHookClusterRole := &rbacv1.ClusterRole{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-webhook-service"}, webHookClusterRole)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), webHookClusterRole, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete serviceAccount", "name", "idp-mgmt-webhook-service", "namespace", podNamespace)
	webHookServiceAccount := &corev1.ServiceAccount{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-webhook-service", Namespace: podNamespace}, webHookServiceAccount)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), webHookServiceAccount, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete Service", "name", "idp-mgmt-webhook-service", "namespace", podNamespace)
	service := &corev1.Service{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-webhook-service", Namespace: podNamespace}, service)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), service, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	r.Log.Info("Delete ValidatingWebhookConfiguration", "name", "idp-mgmt-webhook-service", "namespace", podNamespace)
	validationWebhook := &admissionregistration.ValidatingWebhookConfiguration{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "idp-mgmt-webhook-service", Namespace: podNamespace}, validationWebhook)
	switch {
	case errors.IsNotFound(err):
	case err == nil:
		if err := r.Client.Delete(context.TODO(), validationWebhook, &client.DeleteOptions{}); err != nil {
			return giterrors.WithStack(err)
		}
	default:
		return giterrors.WithStack(err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IDPConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {

	//Check pre-requisite
	if ok, err := r.checkPreRequisite(); !ok {
		return giterrors.WithMessage(err, "IDP prerequisites are not met")
	} else {
		r.Log.Info("IDP prerequisites are met")
	}

	if err := identitatemv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := apiregistrationv1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := admissionregistration.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpoperatorconfig.GetScenarioResourcesReader()

	files := []string{
		"crd/bases/identityconfig.identitatem.io_authrealms.yaml",
		"crd/bases/identityconfig.identitatem.io_idpconfigs.yaml",
		"crd/bases/identityconfig.identitatem.io_strategies.yaml",
	}
	if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", files...); err != nil {
		return giterrors.WithStack(err)
	}

	podName = os.Getenv("POD_NAME")
	podNamespace = os.Getenv("POD_NAMESPACE")
	if len(podName) == 0 || len(podNamespace) == 0 {
		return fmt.Errorf("POD_NAME or POD_NAMESPACE not set")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&identitatemv1alpha1.IDPConfig{}).
		Complete(r)
}

func (r *IDPConfigReconciler) checkPreRequisite() (bool, error) {
	rhacm, rhacmErr := r.isRHACM()
	mce, mceErr := r.isMCE()
	if rhacm || mce {
		r.Log.Info("either Red Hat Advanced Cluster Management or Multicluster Engine is installed")
		return true, nil
	}
	var msg string
	if rhacmErr != nil {
		msg = rhacmErr.Error()
	}
	if mceErr != nil {
		msg = fmt.Sprintf("%s\n%s", msg, mceErr.Error())
	}
	return false, fmt.Errorf("neither Red Hat Advanced Cluster Management or Multicluster Engine installation has been detected, %s", msg)
}

func (r *IDPConfigReconciler) isRHACM() (bool, error) {
	rhacmcr, err := r.getRHACMCRList()
	if err != nil {
		return false, err
	}
	if len(rhacmcr.Items) == 0 {
		return false, fmt.Errorf("the product Red Hat Advanced Cluster Management is not installed on this cluster")
	}
	return true, nil
}

func (r *IDPConfigReconciler) getRHACMCRList() (*unstructured.UnstructuredList, error) {
	dynamicClient := r.DynamicClient
	gvr := schema.GroupVersionResource{Group: "operator.open-cluster-management.io", Version: "v1", Resource: "multiclusterhubs"}
	return dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{})
}

func (r *IDPConfigReconciler) isMCE() (bool, error) {
	cms, err := r.getMCEConfigMapList()
	if err != nil {
		return false, err
	}
	if len(cms.Items) == 0 {
		return false, fmt.Errorf("the product Multicluster Engine is not installed on this cluster")
	}
	return true, nil
}

func (r *IDPConfigReconciler) getMCEConfigMapList() (*unstructured.UnstructuredList, error) {
	dynamicClient := r.DynamicClient
	gvr := schema.GroupVersionResource{Group: "multicluster.openshift.io", Version: "v1", Resource: "multiclusterengines"}
	return dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
}

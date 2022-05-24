// Copyright Red Hat

package clusteroauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	giterrors "github.com/pkg/errors"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ghodss/yaml"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"

	// clusterv1 "open-cluster-management.io/api/cluster/v1"
	// clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	hypershiftdeploymentv1alpha1 "github.com/stolostron/hypershift-deployment-controller/api/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	//+kubebuilder:scaffold:imports
)

func (mgr *HypershiftMgr) Save() (ctrl.Result, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ConfigMapOriginalOAuthName(),
			Namespace: mgr.ClusterOAuth.Namespace,
		},
	}

	//If already exist, do nothing
	mgr.Reconciler.Log.Info("check if configMap already exists",
		"name", helpers.ConfigMapOriginalOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	if err := mgr.Reconciler.Client.Get(context.TODO(),
		client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: mgr.ClusterOAuth.Namespace},
		cm); err == nil {
		return ctrl.Result{}, nil
	}

	hd, err := mgr.getHypershiftDeployment(mgr.ClusterOAuth.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	index := mgr.searchConfiguration(hd, "OAuth", "cluster")
	if index == -1 {
		return ctrl.Result{}, giterrors.WithStack(mgr.Reconciler.Client.Create(context.TODO(), cm))
	}
	b, err := yaml.YAMLToJSON(hd.Spec.HostedClusterSpec.Configuration.Items[index].Raw)
	mgr.Reconciler.Log.Info("OAuth:", "Raw", string(hd.Spec.HostedClusterSpec.Configuration.Items[index].Raw))
	mgr.Reconciler.Log.Info("OAuth:", "Raw-JSON", string(b))
	if err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	cm.Data = map[string]string{
		"json": string(b),
		"yaml": string(hd.Spec.HostedClusterSpec.Configuration.Items[index].Raw),
	}

	return ctrl.Result{}, giterrors.WithStack(mgr.Reconciler.Client.Create(context.TODO(), cm))

}

func (mgr *HypershiftMgr) Push() (err error) {
	// 1. Search the ns on the hub of the hosting cluster
	// search the managedcluster
	mc := &clusterv1.ManagedCluster{}
	err = mgr.Reconciler.Client.Get(context.TODO(), client.ObjectKey{Name: mgr.ClusterOAuth.Namespace}, mc)
	if err != nil {
		return giterrors.WithStack(err)
	}

	// Aggregate All ClusterOAuth singleOAuth
	clusterOAuths := &identitatemv1alpha1.ClusterOAuthList{}
	mgr.Reconciler.Log.Info("search the clusterOAuths in namepsace", "namespace", mgr.ClusterOAuth.GetNamespace())
	if err := mgr.Reconciler.List(context.TODO(),
		clusterOAuths,
		&client.ListOptions{Namespace: mgr.ClusterOAuth.GetNamespace()}); err != nil {
		// Error reading the object - requeue the request.
		return giterrors.WithStack(err)
	}

	singleOAuth := mgr.Reconciler.aggregateClusterOAuths(clusterOAuths, mgr.ClusterOAuth.GetNamespace())
	// create manifest for single OAuth
	data, err := json.Marshal(singleOAuth)
	if err != nil {
		return giterrors.WithStack(err)
	}

	//Search the hypershift deployment
	hd, err := mgr.getHypershiftDeployment(mgr.ClusterOAuth.Namespace)
	if err != nil {
		return err
	}

	index := mgr.searchConfiguration(hd, "OAuth", "cluster")
	if index == -1 {
		hd.Spec.HostedClusterSpec.Configuration.Items =
			append(hd.Spec.HostedClusterSpec.Configuration.Items,
				runtime.RawExtension{Raw: data})
	} else {
		hd.Spec.HostedClusterSpec.Configuration.Items[index] = runtime.RawExtension{Raw: data}
	}

	if len(hd.Spec.HostedClusterSpec.Configuration.SecretRefs) == 0 {
		hd.Spec.HostedClusterSpec.Configuration.SecretRefs = make([]corev1.LocalObjectReference, 0)
	}

	for _, clusterOAuth := range clusterOAuths.Items {
		if clusterOAuth.DeletionTimestamp != nil {
			continue
		}
		mgr.Reconciler.Log.Info(" build clusterOAuth",
			"name: ", clusterOAuth.GetName(),
			"namespace:", clusterOAuth.GetNamespace(),
			"identityProviders:", len(clusterOAuth.Spec.OAuth.Spec.IdentityProviders))
		found := false
		for _, r := range hd.Spec.HostedClusterSpec.Configuration.SecretRefs {
			if r.Name == clusterOAuth.GetNamespace()+"-"+clusterOAuth.Name {
				found = true
				break
			}
		}

		for _, idp := range clusterOAuth.Spec.OAuth.Spec.IdentityProviders {

			//Look for secret for Identity Provider and if found, add to manifest work
			secret := &corev1.Secret{}

			mgr.Reconciler.Log.Info("retrieving client secret", "name", clusterOAuth.Name, "namespace", clusterOAuth.Namespace)
			if err := mgr.Reconciler.Client.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterOAuth.Namespace, Name: clusterOAuth.Name},
				secret); err != nil {
				return giterrors.WithStack(err)
			}
			//add secret to manifest

			newSecret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterOAuth.GetNamespace() + "-" + secret.Name,
					Namespace: hd.Namespace,
				},
			}

			newSecret.Data = secret.Data
			newSecret.Type = secret.Type

			mgr.Reconciler.Log.Info("add secret to clusters ns for idp", "name", idp.Name)

			if err := mgr.Reconciler.Client.Create(context.TODO(), newSecret, &client.CreateOptions{}); err != nil {
				if !errors.IsAlreadyExists(err) {
					return giterrors.WithStack(err)
				}
				if err = mgr.Reconciler.Client.Update(context.TODO(), newSecret, &client.UpdateOptions{}); err != nil {
					return giterrors.WithStack(err)
				}
			}
			if !found {
				hd.Spec.HostedClusterSpec.Configuration.SecretRefs =
					append(hd.Spec.HostedClusterSpec.Configuration.SecretRefs,
						corev1.LocalObjectReference{Name: newSecret.Name})
			}
		}
	}

	mgr.Reconciler.Log.Info("Update hd for", "cluster", mgr.ClusterOAuth.GetNamespace())
	hd.GetLabels()[helpers.HypershiftDeploymentForceReconcileLabel] =
		strings.ReplaceAll(metav1.Now().Format(time.RFC3339), ":", "-")
	if err := mgr.Reconciler.Client.Update(context.TODO(), hd, &client.UpdateOptions{}); err != nil {
		mgr.Reconciler.Log.Error(err, "Failed to update hc for component")
		return giterrors.WithStack(err)
	}
	return nil
}

func (mgr *HypershiftMgr) getHypershiftDeployment(clusterName string) (*hypershiftdeploymentv1alpha1.HypershiftDeployment, error) {
	return helpers.GetHypershiftDeployment(mgr.Reconciler.Client, clusterName)
}

// unmanagedHypershiftCluster deletes a manifestwork
func (mgr *HypershiftMgr) Unmanage() (ctrl.Result, error) {
	clusterOAuths := &identitatemv1alpha1.ClusterOAuthList{}
	if err := mgr.Reconciler.Client.List(context.TODO(),
		clusterOAuths,
		client.InNamespace(mgr.ClusterOAuth.Namespace)); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	if len(clusterOAuths.Items) == 1 {
		hd, err := mgr.getHypershiftDeployment(mgr.ClusterOAuth.Namespace)
		if err != nil {
			return ctrl.Result{}, err
		}
		index := mgr.searchConfiguration(hd, "OAuth", "cluster")
		//No OAuth to reset
		if index == -1 {
			return ctrl.Result{}, nil
		}

		oauth, err := mgr.Reconciler.getOriginalOAuth(mgr.ClusterOAuth)
		switch {
		case err == nil:
			items := hd.Spec.HostedClusterSpec.Configuration.Items
			if v, ok := oauth.Data["json"]; ok {
				items[index] = runtime.RawExtension{Raw: []byte(v)}
			} else {
				items[index] = hd.Spec.HostedClusterSpec.Configuration.Items[len(items)-1]
				items = items[:len(items)-1]
			}
			hd.Spec.HostedClusterSpec.Configuration.Items = items
			mgr.Reconciler.Log.Info("restore hd oauth for", "hd", hd.Name)
			if err := mgr.Reconciler.Client.Update(context.TODO(), hd, &client.UpdateOptions{}); err != nil {
				mgr.Reconciler.Log.Error(err, "Failed to restore hd", "hd", hd.Name)
				return ctrl.Result{}, giterrors.WithStack(err)
			}
			if err := mgr.Reconciler.deleteOriginalOAuth(mgr.ClusterOAuth.Namespace); err != nil {
				return ctrl.Result{}, err
			}
		case errors.IsNotFound(err):
			mgr.Reconciler.Log.Info("WARNING: original oauth not found, can not restore",
				"name", mgr.ClusterOAuth.Name,
				"namespace", mgr.ClusterOAuth.Namespace)
			return ctrl.Result{}, nil
		default:
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		return ctrl.Result{}, nil
	}

	if err := mgr.Push(); err != nil {
		cond := metav1.Condition{
			Type:   identitatemv1alpha1.ClusterOAuthApplied,
			Status: metav1.ConditionFalse,
			Reason: "updateHypershiftDeploymentFailed",
			Message: fmt.Sprintf("ClusterOAuth %s failed to update HypershiftDeployment for cluster %s Error:, %s",
				mgr.ClusterOAuth.Name, mgr.ClusterOAuth.Namespace, err.Error()),
		}

		if err := mgr.Reconciler.setConditions(mgr.ClusterOAuth, cond); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (mgr *HypershiftMgr) searchConfiguration(hd *hypershiftdeploymentv1alpha1.HypershiftDeployment,
	kind, name string) int {
	if hd.Spec.HostedClusterSpec.Configuration == nil {
		hd.Spec.HostedClusterSpec.Configuration = &hypershiftv1alpha1.ClusterConfiguration{}
		hd.Spec.HostedClusterSpec.Configuration.Items = make([]runtime.RawExtension, 0)
		return -1
	}
	switch kind {
	case "OAuth":
		for i, re := range hd.Spec.HostedClusterSpec.Configuration.Items {
			existing := &openshiftconfigv1.OAuth{}
			err := json.Unmarshal(re.Raw, existing)
			if err != nil {
				continue
			}
			if existing.GetName() == name {
				return i
			}
		}
	case "ClusterRole":
		for i, re := range hd.Spec.HostedClusterSpec.Configuration.Items {
			existing := &rbacv1.ClusterRole{}
			err := json.Unmarshal(re.Raw, existing)
			if err != nil {
				continue
			}
			if existing.GetName() == name {
				return i
			}
		}
	}
	return -1
}

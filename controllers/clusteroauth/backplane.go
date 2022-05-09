// Copyright Red Hat

package clusteroauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	giterrors "github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ghodss/yaml"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	"github.com/identitatem/idp-mgmt-operator/resources"

	// clusterv1 "open-cluster-management.io/api/cluster/v1"
	// clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	manifestworkv1 "open-cluster-management.io/api/work/v1"

	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	idpmgmtconfig "github.com/identitatem/idp-mgmt-operator/config"

	//+kubebuilder:scaffold:imports

	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

func (mgr *BackplaneMgr) Save() (mgresult ctrl.Result, err error) {
	cm := &corev1.ConfigMap{}

	//If already exist, do nothing
	mgr.Reconciler.Log.Info("check if configMap already exists",
		"name", helpers.ConfigMapOriginalOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	if err := mgr.Reconciler.Get(context.TODO(), client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: mgr.ClusterOAuth.Namespace}, cm); err == nil {
		return ctrl.Result{}, nil
	}

	mgr.Reconciler.Log.Info("create managedclusterview",
		"name", helpers.ManagedClusterViewOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(mgr.Reconciler.KubeClient,
		mgr.Reconciler.APIExtensionClient,
		mgr.Reconciler.DynamicClient).Build()

	readerResources := resources.GetScenarioResourcesReader()

	file := "managedclusterview/managed-cluster-view-oauth.yaml"

	values := struct {
		Name      string
		Namespace string
	}{
		Name:      helpers.ManagedClusterViewOAuthName(),
		Namespace: mgr.ClusterOAuth.Namespace,
	}
	if _, err := applier.ApplyCustomResources(readerResources, values, false, "", file); err != nil {
		return ctrl.Result{}, err
	}
	mgr.Reconciler.Log.Info("saveManagedClusterViewForOAuthResult for",
		"name", mgr.ClusterOAuth.Name,
		"namespace", mgr.ClusterOAuth.Namespace)
	return mgr.saveManagedClusterViewForOAuthResult()
}

func (mgr *BackplaneMgr) saveManagedClusterViewForOAuthResult() (mgresult ctrl.Result, err error) {
	mcvOAuth := &viewv1beta1.ManagedClusterView{}
	mgr.Reconciler.Log.Info("check if managedclusterview exists",
		"name", helpers.ManagedClusterViewOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	if err := mgr.Reconciler.Get(context.TODO(),
		client.ObjectKey{Name: helpers.ManagedClusterViewOAuthName(), Namespace: mgr.ClusterOAuth.Namespace},
		mcvOAuth); err != nil {
		return ctrl.Result{}, err
	}

	mgr.Reconciler.Log.Info("check if managedclusterview has results",
		"name", helpers.ManagedClusterViewOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	if len(mcvOAuth.Status.Result.Raw) == 0 {
		mgr.Reconciler.Log.Info("waiting for original oauth", "cluster", mgr.ClusterOAuth.Namespace)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	mgr.Reconciler.Log.Info("create configmap containing the OAuth",
		"name", helpers.ConfigMapOriginalOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	b, err := yaml.JSONToYAML(mcvOAuth.Status.Result.Raw)
	if err != nil {
		return ctrl.Result{}, err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ConfigMapOriginalOAuthName(),
			Namespace: mgr.ClusterOAuth.Namespace,
		},
		Data: map[string]string{
			"json": string(mcvOAuth.Status.Result.Raw),
			"yaml": string(b),
		},
	}

	if err := mgr.Reconciler.Create(context.TODO(), cm); err != nil {
		return ctrl.Result{}, err
	}

	mgr.Reconciler.Log.Info("delete managedclusterview",
		"name", helpers.ManagedClusterViewOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	if err := mgr.Reconciler.Delete(context.TODO(), mcvOAuth); err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (mgr *BackplaneMgr) Push() (err error) {
	// Create empty manifest work
	mgr.Reconciler.Log.Info("prepare manifestwork",
		"name", helpers.ManifestWorkOAuthName(),
		"namespace", mgr.ClusterOAuth.Namespace)
	manifestWorkOAuth := &manifestworkv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ManifestWorkOAuthName(),
			Namespace: mgr.ClusterOAuth.Namespace,
			Annotations: map[string]string{
				posthookAnnotation: "60",
			},
		},
		Spec: manifestworkv1.ManifestWorkSpec{
			DeleteOption: &manifestworkv1.DeleteOption{
				PropagationPolicy: manifestworkv1.DeletePropagationPolicyTypeOrphan,
			},
			Workload: manifestworkv1.ManifestsTemplate{
				Manifests: []manifestworkv1.Manifest{},
			},
		},
	}
	//Add Aggregated role
	err = mgr.addAggregatedRoleToManifestwork(manifestWorkOAuth)
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

	for _, clusterOAuth := range clusterOAuths.Items {
		//build Secrets and add to manifest work
		if clusterOAuth.DeletionTimestamp != nil {
			continue
		}
		mgr.Reconciler.Log.Info(" build clusterOAuth",
			"name: ", clusterOAuth.GetName(),
			"namespace:", clusterOAuth.GetNamespace(),
			"identityProviders:", len(clusterOAuth.Spec.OAuth.Spec.IdentityProviders))

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
					Namespace: "openshift-config",
				},
			}

			newSecret.Data = secret.Data
			newSecret.Type = secret.Type

			data, err := json.Marshal(newSecret)
			if err != nil {
				return giterrors.WithStack(err)
			}

			manifest := manifestworkv1.Manifest{
				RawExtension: runtime.RawExtension{Raw: data},
			}

			//add manifest to manifest work
			mgr.Reconciler.Log.Info("append workload for secret in manifest for idp", "name", idp.Name)
			manifestWorkOAuth.Spec.Workload.Manifests = append(manifestWorkOAuth.Spec.Workload.Manifests, manifest)

		}
	}

	// create manifest for single OAuth
	data, err := json.Marshal(singleOAuth)
	if err != nil {
		return giterrors.WithStack(err)
	}

	manifest := manifestworkv1.Manifest{
		RawExtension: runtime.RawExtension{Raw: data},
	}

	//add OAuth manifest to manifest work
	mgr.Reconciler.Log.Info("append workload for oauth in manifest")
	manifestWorkOAuth.Spec.Workload.Manifests = append(manifestWorkOAuth.Spec.Workload.Manifests, manifest)

	// create manifest work for managed cluster
	if err := mgr.CreateOrUpdateManifestWork(manifestWorkOAuth); err != nil {
		mgr.Reconciler.Log.Error(err, "Failed to create manifest work for component")
		// Error reading the object - requeue the request.
		return giterrors.WithStack(err)
	}
	return nil
}

func (mgr *BackplaneMgr) addAggregatedRoleToManifestwork(mw *manifestworkv1.ManifestWork) error {
	aggregatedRoleJson, err := mgr.Reconciler.generateAggregatedRole()
	if err != nil {
		return giterrors.WithStack(err)
	}

	manifestAggregated := manifestworkv1.Manifest{
		RawExtension: runtime.RawExtension{Raw: aggregatedRoleJson},
	}

	mw.Spec.Workload.Manifests = append(mw.Spec.Workload.Manifests, manifestAggregated)

	return nil
}

// CreateOrUpdateManifestWork creates a new ManifestWork or update an existing ManifestWork
func (mgr *BackplaneMgr) CreateOrUpdateManifestWork(manifestwork *manifestworkv1.ManifestWork) error {

	oldManifestwork := &manifestworkv1.ManifestWork{}

	err := mgr.Reconciler.Get(
		context.TODO(),
		types.NamespacedName{Name: manifestwork.Name, Namespace: manifestwork.Namespace},
		oldManifestwork,
	)
	if err == nil {
		oldManifestwork.Spec.Workload.Manifests = manifestwork.Spec.Workload.Manifests
		if err := mgr.Reconciler.Update(context.TODO(), oldManifestwork); err != nil {
			mgr.Reconciler.Log.Error(err, "Fail to update manifestwork")
			return giterrors.WithStack(err)
		}
		return nil
	}
	if errors.IsNotFound(err) {
		mgr.Reconciler.Log.Info("create manifestwork", "name", manifestwork.Name, "namespace", manifestwork.Namespace)
		if err := mgr.Reconciler.Create(context.TODO(), manifestwork); err != nil {
			mgr.Reconciler.Log.Error(err, "Fail to create manifestwork")
			return giterrors.WithStack(err)
		}
		return nil
	}
	return err
}

// unmanagedBackplaneCluster deletes a manifestwork
func (mgr *BackplaneMgr) Unmanage() (mgresult ctrl.Result, err error) {
	clusterOAuths := &identitatemv1alpha1.ClusterOAuthList{}
	if err := mgr.Reconciler.List(context.TODO(),
		clusterOAuths,
		client.InNamespace(mgr.ClusterOAuth.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	if len(clusterOAuths.Items) == 1 {
		mgr.Reconciler.Log.Info("delete for manifestwork",
			"name", helpers.ManifestWorkOAuthName(),
			"namespace", mgr.ClusterOAuth.Namespace)
		if err := mgr.deleteManifestWork(helpers.ManifestWorkOAuthName(), mgr.ClusterOAuth.Namespace); err != nil {
			return ctrl.Result{}, err
		}
		err := mgr.restoreOriginalOAuth()
		switch {
		case err == nil:
			if err := mgr.checkManifestWorkOriginalOAuthApplied(mgr.ClusterOAuth.Namespace); err != nil {
				return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
			}
			if err := mgr.Reconciler.deleteOriginalOAuth(mgr.ClusterOAuth.Namespace); err != nil {
				return ctrl.Result{}, err
			}
		case errors.IsNotFound(err):
		default:
			return ctrl.Result{}, err
		}
		if err := mgr.deleteManifestWork(helpers.ManifestWorkOriginalOAuthName(), mgr.ClusterOAuth.Namespace); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if err := mgr.Push(); err != nil {
		cond := metav1.Condition{
			Type:   identitatemv1alpha1.ClusterOAuthApplied,
			Status: metav1.ConditionFalse,
			Reason: "CreateManifestWorkFailed",
			Message: fmt.Sprintf("ClusterOAuth %s failed to create ManifestWork for cluster %s Error:, %s",
				mgr.ClusterOAuth.Name, mgr.ClusterOAuth.Namespace, err.Error()),
		}

		if err := mgr.Reconciler.setConditions(mgr.ClusterOAuth, cond); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (mgr *BackplaneMgr) restoreOriginalOAuth() (err error) {
	originalOAuth, err := mgr.Reconciler.getOriginalOAuth(mgr.ClusterOAuth)
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		mgr.Reconciler.Log.Info("WARNING: original oauth not found, can not restore",
			"name", mgr.ClusterOAuth.Name, "namespace", mgr.ClusterOAuth.Namespace)
		return err
	default:
		return err
	}

	jsonData, err := yaml.YAMLToJSON([]byte(originalOAuth.Data["json"]))
	if err != nil {
		return err
	}
	mw := &manifestworkv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ManifestWorkOriginalOAuthName(),
			Namespace: mgr.ClusterOAuth.GetNamespace(),
			Annotations: map[string]string{
				posthookAnnotation: "60",
			},
		},
		Spec: manifestworkv1.ManifestWorkSpec{
			DeleteOption: &manifestworkv1.DeleteOption{
				PropagationPolicy: manifestworkv1.DeletePropagationPolicyTypeOrphan,
			},
			Workload: manifestworkv1.ManifestsTemplate{
				Manifests: []manifestworkv1.Manifest{
					{RawExtension: runtime.RawExtension{Raw: jsonData}},
				},
			},
		},
	}

	//Add Aggregated role
	mgr.Reconciler.Log.Info("add aggregated role")
	aggregatedRoleYaml, err := idpmgmtconfig.GetScenarioResourcesReader().Asset("rbac/role-aggregated-clusterrole.yaml")
	if err != nil {
		return err
	}

	aggregatedRoleJson, err := yaml.YAMLToJSON(aggregatedRoleYaml)
	if err != nil {
		return err
	}

	manifestAggregated := manifestworkv1.Manifest{
		RawExtension: runtime.RawExtension{Raw: aggregatedRoleJson},
	}

	mw.Spec.Workload.Manifests = append(mw.Spec.Workload.Manifests, manifestAggregated)

	if err := mgr.CreateOrUpdateManifestWork(mw); err != nil {
		return err
	}

	return nil
}

func (mgr *BackplaneMgr) checkManifestWorkOriginalOAuthApplied(ns string) error {
	manifestWork := &manifestworkv1.ManifestWork{}
	if err := mgr.Reconciler.Get(
		context.TODO(),
		types.NamespacedName{Name: helpers.ManifestWorkOriginalOAuthName(), Namespace: ns},
		manifestWork,
	); err != nil {
		return giterrors.WithStack(err)
	}
	for _, c := range manifestWork.Status.Conditions {
		if c.Type == string(manifestworkv1.ManifestApplied) &&
			c.Status == metav1.ConditionTrue {
			return nil
		}
	}
	return fmt.Errorf("manifestwork %s not yet Applied", helpers.ManifestWorkOriginalOAuthName())
}

func (mgr *BackplaneMgr) deleteManifestWork(name, ns string) error {
	manifestWork := &manifestworkv1.ManifestWork{}
	if err := mgr.Reconciler.Get(
		context.TODO(),
		types.NamespacedName{Name: name, Namespace: ns},
		manifestWork,
	); err != nil {
		if !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
		return nil
	}
	if manifestWork.DeletionTimestamp.IsZero() {
		mgr.Reconciler.Log.Info("delete manifest", "name", manifestWork.Name, "namespace", manifestWork.Namespace)
		err := mgr.Reconciler.Delete(context.TODO(), manifestWork)
		if err != nil && !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
	}
	return nil
}

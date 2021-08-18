// Copyright Red Hat

package main

import (
	"flag"
	"os"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	authrealmv1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	idpmgmtoperatorconfig "github.com/identitatem/idp-mgmt-operator/config"
	"github.com/identitatem/idp-mgmt-operator/controllers"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = authrealmv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "628f2987.identitatem.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	r := &controllers.AuthRealmReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Log:                ctrl.Log.WithName("controllers").WithName("AuthRealm"),
		Scheme:             mgr.GetScheme(),
	}
	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AuthRealm")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpmgmtoperatorconfig.GetScenarioResourcesReader()

	file := "crd/bases/identityconfig.identitatem.io_authrealms.yaml"
	_, err = applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", file)
	if err != nil {
		setupLog.Error(err, "unable to create install the crd for controller", "crd", file, "controller", "AuthRealm")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

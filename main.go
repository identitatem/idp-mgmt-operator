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

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"

	"github.com/identitatem/idp-mgmt-operator/controllers/authrealm"
	"github.com/identitatem/idp-mgmt-operator/controllers/clusteroauth"
	"github.com/identitatem/idp-mgmt-operator/controllers/placementdecision"
	"github.com/identitatem/idp-mgmt-operator/controllers/strategy"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = identitatemv1alpha1.AddToScheme(scheme)
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

	if err = (&authrealm.AuthRealmReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Log:                ctrl.Log.WithName("controllers").WithName("AuthRealm"),
		Scheme:             mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AuthRealm")
		os.Exit(1)
	}

	//This manager is in charge of creating a Placement per strategy
	//based on the strategy and authrealm
	if err = (&strategy.StrategyReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Scheme:             mgr.GetScheme(),
		Log:                ctrl.Log.WithName("controllers").WithName("Strategy"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Strategy")
		os.Exit(1)
	}

	//This manager creates the DexClient and ClusterOAuth based on placementDecision
	if err = (&placementdecision.PlacementDecisionReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Scheme:             mgr.GetScheme(),
		Log:                ctrl.Log.WithName("controllers").WithName("PlacementDecision"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PlacementDecision")
		os.Exit(1)
	}

	//This manager consolidate all ClusterOAuth into one OAUth and
	//send it to the managedcluster using the available strategy
	if err = (&clusteroauth.ClusterOAuthReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Scheme:             mgr.GetScheme(),
		Log:                ctrl.Log.WithName("controllers").WithName("ClusterOAuth"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterOAuth")
		os.Exit(1)
	}

	if err = (&identitatemv1alpha1.AuthRealm{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AuthRealm")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

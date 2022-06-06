// Copyright Red Hat

package manager

import (
	"context"
	"os"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/controllers/authrealm"
	"github.com/identitatem/idp-mgmt-operator/controllers/clusteroauth"
	"github.com/identitatem/idp-mgmt-operator/controllers/hypershiftdeployment"
	"github.com/identitatem/idp-mgmt-operator/controllers/manifestwork"
	"github.com/identitatem/idp-mgmt-operator/controllers/placement"
	"github.com/identitatem/idp-mgmt-operator/controllers/strategy"
	"github.com/spf13/cobra"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type managerOptions struct {
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
}

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = identitatemv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func NewManager() *cobra.Command {
	o := &managerOptions{}
	cmd := &cobra.Command{
		Use:   "manager",
		Short: "manager for idp-mgmt-operator",
		Run: func(cmd *cobra.Command, args []string) {
			o.run()
			os.Exit(1)
		},
	}
	cmd.Flags().StringVar(&o.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().StringVar(&o.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	cmd.Flags().BoolVar(&o.enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	return cmd
}

func (o *managerOptions) run() {

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	setupLog.Info("Setup Manager")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     o.metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: o.probeAddr,
		LeaderElection:         o.enableLeaderElection,
		LeaderElectionID:       "628f2987.identitatem.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("Add AuthRealm reconciler")

	hypershiftDeploymentInstalled := true
	apiExtensionsClient := apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie())
	_, err = apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(),
		"hypershiftdeployments.cluster.open-cluster-management.io",
		metav1.GetOptions{})
	if err != nil {
		setupLog.Info("hypershiftdeployments.cluster.open-cluster-management.io CRD", "err", err)
		if !errors.IsNotFound(err) {
			setupLog.Error(err, "unable to check HypershiftDeployment CRD")
			os.Exit(1)
		}
		hypershiftDeploymentInstalled = false
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
	setupLog.Info("Add Strategy reconciler")
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
	setupLog.Info("Add DexClient reconciler")
	if err = (&placement.PlacementReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Scheme:             mgr.GetScheme(),
		Log:                ctrl.Log.WithName("controllers").WithName("Placement"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Placement")
		os.Exit(1)
	}

	//This manager consolidate all ClusterOAuth into one OAUth and
	//send it to the managedcluster using the available strategy
	setupLog.Info("Add ClusterOAuth reconciler")
	if err = (&clusteroauth.ClusterOAuthReconciler{
		Client:                        mgr.GetClient(),
		KubeClient:                    kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:                 dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient:            apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		HypershiftDeploymentInstalled: hypershiftDeploymentInstalled,
		Scheme:                        mgr.GetScheme(),
		Log:                           ctrl.Log.WithName("controllers").WithName("ClusterOAuth"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterOAuth")
		os.Exit(1)
	}

	//This manager consolidate all ManifestWork to update the Authrealm status
	setupLog.Info("Add ManifestWork reconciler")
	if err = (&manifestwork.ManifestWorkReconciler{
		Client:             mgr.GetClient(),
		KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Scheme:             mgr.GetScheme(),
		Log:                ctrl.Log.WithName("controllers").WithName("ManifestWork"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ManifestWork")
		os.Exit(1)
	}

	//This manager consolidate all HypershiftDeployment to update the Authrealm status
	if hypershiftDeploymentInstalled {
		setupLog.Info("Add HypershiftDeployment reconciler")
		if err = (&hypershiftdeployment.HypershiftDeploymentReconciler{
			Client:             mgr.GetClient(),
			KubeClient:         kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()),
			DynamicClient:      dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
			APIExtensionClient: apiextensionsclient.NewForConfigOrDie(ctrl.GetConfigOrDie()),
			Scheme:             mgr.GetScheme(),
			Log:                ctrl.Log.WithName("controllers").WithName("HypershiftDeployment"),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "HypershiftDeployment")
			os.Exit(1)
		}
	}

	// add healthz/readyz check handler
	setupLog.Info("Add health check")
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to add healthz check handler ")
		os.Exit(1)
	}

	setupLog.Info("Add ready check")
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to add readyz check handler ")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

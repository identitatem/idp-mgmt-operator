// Copyright Red Hat

// +build e2e

package utils

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type TestOptionsType struct {
	HubCluster      Cluster         `yaml:"hub"`
	ManagedClusters []Cluster       `yaml:"managedclusters"`
}

// Defines the shape of the hub cluster and its managed clusters
type Cluster struct {
	ClusterServerURL	string								`yaml:"clusterServerUrl,omitempty"`
	KubeConfig      	string          					`yaml:"kubeConfig,omitempty"`
	KubeContext			string								`yaml:"kubeContext,omitempty"`
	KubeClient 			kubernetes.Interface				`yaml:"kubeClient,omitempty"`
	KubeClientDynamic	dynamic.Interface					`yaml:"kubeClientDynamic,omitempty"`
	ApiExtensionsClient	apiextensionsclientset.Interface 	`yaml:"apiExtensionsClient,omitempty"` 
}

// Get GVR for a resource kind
func GetGVRForResource (resourceKind string) (schema.GroupVersionResource, error) {
	switch resourceKind {
	case "ManagedClusterSet":
		return schema.GroupVersionResource{
			Group:    "cluster.open-cluster-management.io",
			Version:  "v1beta1",
			Resource: "managedclustersets"}, nil
	case "ManagedClusterSetBinding":
		return schema.GroupVersionResource{
			Group:    "cluster.open-cluster-management.io",
			Version:  "v1beta1",
			Resource: "managedclustersetbindings"}, nil
	case "Placement":
		return schema.GroupVersionResource{
			Group:    "cluster.open-cluster-management.io",
			Version:  "v1alpha1",
			Resource: "placements"}, nil
	case "AuthRealm":
		return schema.GroupVersionResource{
			Group:    "identityconfig.identitatem.io",
			Version:  "v1alpha1",
			Resource: "authrealms"}, nil
	case "Strategy":
		return schema.GroupVersionResource{
			Group:    "identityconfig.identitatem.io",
			Version:  "v1alpha1",
			Resource: "strategies"}, nil
	case "ClusterOAuth":
		return schema.GroupVersionResource{
			Group:    "identityconfig.identitatem.io",
			Version:  "v1alpha1",
			Resource: "clusteroauths"}, nil		
	case "PlacementDecision":
		return schema.GroupVersionResource{
			Group:    "cluster.open-cluster-management.io",
			Version:  "v1alpha1",
			Resource: "placementdecisions"}, nil
	case "DexServer":
		return schema.GroupVersionResource{
			Group:    "auth.identitatem.io",
			Version:  "v1alpha1",
			Resource: "dexservers"}, nil
	case "DexClient":
		return schema.GroupVersionResource{
			Group:    "auth.identitatem.io",
			Version:  "v1alpha1",
			Resource: "dexclients"}, nil
	case "ManifestWork":
		return schema.GroupVersionResource{
			Group:    "work.open-cluster-management.io",
			Version:  "v1",
			Resource: "manifestworks"}, nil
	case "ManagedCluster":
		return schema.GroupVersionResource{
			Group:    "cluster.open-cluster-management.io",
			Version:  "v1",
			Resource: "managedclusters"}, nil
	case "OAuth":
		return schema.GroupVersionResource{
			Group:    "config.openshift.io",
			Version:  "v1",
			Resource: "oauths"}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("resource %s not supported", resourceKind)
	}
}

// Returns the GVR for ManagedClusterSet
func NewManagedClusterSetGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1beta1",
		Resource: "managedclustersets"}
}

// Returns the GVR for ManagedClusterSetBinding
func NewManagedClusterSetBindingGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1beta1",
		Resource: "managedclustersetbindings"}
}

// Returns the GVR for Placement
func NewPlacementGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "placements"}
}

// Returns the GVR for AuthRealm
func NewAuthRealmGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "identityconfig.identitatem.io",
		Version:  "v1alpha1",
		Resource: "authrealms"}
}

// Returns the GVR for Strategy
func NewStrategyGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "identityconfig.identitatem.io",
		Version:  "v1alpha1",
		Resource: "strategies"}
}

func LoadConfig(url, kubeconfig, ctx string) (*rest.Config, error) {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	klog.V(5).Infof("Kubeconfig path %s\n", kubeconfig)
	// If we have an explicit indication of where the kubernetes config lives, read that.
	if kubeconfig != "" {
		if ctx == "" {
			// klog.V(5).Infof("clientcmd.BuildConfigFromFlags with %s and %s", url, kubeconfig)
			return clientcmd.BuildConfigFromFlags(url, kubeconfig)
		} else {
			return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
				&clientcmd.ConfigOverrides{
					CurrentContext: ctx,
				}).ClientConfig()
		}
	}
	// If not, try the in-cluster config.
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory.
	if usr, err := user.Current(); err == nil {
		klog.V(5).Infof("clientcmd.BuildConfigFromFlags for url %s using %s\n",
			url,
			filepath.Join(usr.HomeDir, ".kube", "config"))
		if c, err := clientcmd.BuildConfigFromFlags(url, filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not create a valid kubeconfig")
}

func NewKubeClient(url, kubeconfig, ctx string) kubernetes.Interface {
	klog.V(5).Infof("Create kubeclient for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, ctx)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func NewKubeClientDynamic(url, kubeconfig, ctx string) dynamic.Interface {
	klog.V(5).Infof("Create kubeclient dynamic for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, ctx)
	if err != nil {
		panic(err)
	}

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func NewKubeClientAPIExtension(url, kubeconfig, ctx string) apiextensionsclientset.Interface {
	klog.V(5).Infof("Create kubeclient apiextension for url %s using kubeconfig path %s\n", url, kubeconfig)
	config, err := LoadConfig(url, kubeconfig, ctx)
	if err != nil {
		panic(err)
	}

	clientset, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

// Apply a multi resources file to the cluster described by the provided kube client.
// yamlB is a byte array containing the resources file
func Apply(clientKube kubernetes.Interface, clientDynamic dynamic.Interface, yamlB []byte) error {
	// Replace values of environment variables for GitHub client ID and Secret
	yamlsString := string(yamlB)
	yamlsString = strings.Replace(yamlsString, "$GITHUB_CLIENT_ID", os.Getenv("GITHUB_CLIENT_ID"), -1)
	yamlsString = strings.Replace(yamlsString, "$GITHUB_CLIENT_SECRET", os.Getenv("GITHUB_CLIENT_SECRET"), -1)

	yamls := strings.Split(yamlsString, "---")
	// yamls is an []string
	for _, f := range yamls {
		if len(strings.TrimSpace(f)) == 0 {
			continue
		}

		obj := &unstructured.Unstructured{}
		klog.V(5).Infof("obj:%v\n", obj.Object)
		err := yaml.Unmarshal([]byte(f), obj)
		if err != nil {
			return err
		}

		var kind string
		if v, ok := obj.Object["kind"]; !ok {
			return fmt.Errorf("kind attribute not found in %s", f)
		} else {
			kind = v.(string)
		}

		klog.V(5).Infof("kind: %s\n", kind)

		var apiVersion string
		if v, ok := obj.Object["apiVersion"]; !ok {
			return fmt.Errorf("apiVersion attribute not found in %s", f)
		} else {
			apiVersion = v.(string)
		}
		klog.V(5).Infof("apiVersion: %s\n", apiVersion)

		// now use switch over the type of the object
		// and match each type-case
		switch kind {
		case "Namespace":
			klog.V(5).Infof("Install %s: %s\n", kind, f)
			obj := &corev1.Namespace{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			existingObject, errGet := clientKube.CoreV1().
				Namespaces().
				Get(context.TODO(), obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().Namespaces().Create(context.TODO(), obj, metav1.CreateOptions{})
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s already exists, updating!", obj.Kind, obj.Name)
				_, err = clientKube.CoreV1().Namespaces().Update(context.TODO(), existingObject, metav1.UpdateOptions{})
			}
		case "Secret":
			klog.V(5).Infof("Install %s\n", kind)
			obj := &corev1.Secret{}
			err = yaml.Unmarshal([]byte(f), obj)
			if err != nil {
				return err
			}
			
			existingObject, errGet := clientKube.CoreV1().
				Secrets(obj.Namespace).
				Get(context.TODO(), obj.Name, metav1.GetOptions{})
			if errGet != nil {
				_, err = clientKube.CoreV1().Secrets(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
			} else {
				obj.ObjectMeta = existingObject.ObjectMeta
				klog.Warningf("%s %s/%s already exists, updating!", obj.Kind, obj.Namespace, obj.Name)
				_, err = clientKube.CoreV1().Secrets(obj.Namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
			}	
		default:
			gvr, err := GetGVRForResource(kind)

			if err != nil {
				return fmt.Errorf("resource %s not supported", kind)
			}

			klog.V(5).Infof("Install %s\n", kind)

			objName := obj.GetName()
			klog.V(5).Infof("Install %s\n", objName)

			if ns := obj.GetNamespace(); ns != "" {
				existingObject, errGet := clientDynamic.Resource(gvr).
					Namespace(ns).
					Get(context.TODO(), obj.GetName(), metav1.GetOptions{})

				if errGet != nil {
					_, err = clientDynamic.Resource(gvr).
						Namespace(ns).
						Create(context.TODO(), obj, metav1.CreateOptions{})
				} else {
					obj.Object["metadata"] = existingObject.Object["metadata"]
					klog.Warningf("%s %s/%s already exists, updating!", obj.GetKind(), obj.GetNamespace(), obj.GetName())
					_, err = clientDynamic.Resource(gvr).Namespace(ns).Update(context.TODO(), obj, metav1.UpdateOptions{})
				}
			} else {
				existingObject, errGet := clientDynamic.Resource(gvr).Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
				if errGet != nil {
					_, err = clientDynamic.Resource(gvr).Create(context.TODO(), obj, metav1.CreateOptions{})
				} else {
					obj.Object["metadata"] = existingObject.Object["metadata"]
					klog.Warningf("%s %s already exists, updating!", obj.GetKind(), obj.GetName())
					_, err = clientDynamic.Resource(gvr).Update(context.TODO(), obj, metav1.UpdateOptions{})
				}
			}	
		}

		if err != nil {
			return err
		}
	}
	return nil
}

package client

import (
	"os"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig checks if the KUBECONFIG environment variable is set; if not, use the default.
func GetKubeConfig() string {
	env, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		return os.ExpandEnv("$HOME/.kube/config")
	}
	return env
}

// NewKubeClient gets the Kubernetes config and returns a clientset.
func NewKubeClient() (kubernetes.Interface, error) {
	var config *rest.Config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Info().Msg("In-cluster config not available, looking for kubeconfig")
		config, err = clientcmd.BuildConfigFromFlags("", GetKubeConfig())
		if err != nil {
			log.Error().Err(err).Msg("Failed to build config from kubeconfig")
			return nil, err
		}
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Kubernetes client")
		return nil, err
	}
	return client, nil
}

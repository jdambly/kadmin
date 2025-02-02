package drain

import (
	"context"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
)

func DrainNode(clientset kubernetes.Interface, nodeName string) error {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	helper := &drain.Helper{
		Ctx:                 context.TODO(),
		Client:              clientset,
		Force:               true,
		IgnoreAllDaemonSets: true,
		DeleteEmptyDirData:  true,
		Out:                 os.Stdout,
		ErrOut:              os.Stderr,
	}

	return drain.RunCordonOrUncordon(helper, node, true)
}

func UncordonNode(clientset kubernetes.Interface, nodeName string) error {
	log.Info().Str("node", nodeName).Msg("Uncordoning node")
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	helper := &drain.Helper{
		Ctx:    context.TODO(),
		Client: clientset,
	}

	return drain.RunCordonOrUncordon(helper, node, false)
}

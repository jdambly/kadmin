package node

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WaitForPodsReady(clientset kubernetes.Interface, nodeName string) error {
	log.Info().Str("node", nodeName).Msg("Waiting for pods to be ready")
	for {
		pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
		})
		if err != nil {
			return err
		}

		allReady := true
		for _, pod := range pods.Items {
			// skip pods that are completed
			if pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
			if pod.Status.Phase != corev1.PodRunning {
				allReady = false
				break
			}
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}
		}

		if allReady {
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}

func AnnotateNode(clientset kubernetes.Interface, nodeName, key, value string) error {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}
	node.Annotations[key] = value

	_, err = clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	return err
}

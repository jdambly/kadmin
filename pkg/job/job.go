package job

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"time"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateNodeJob(clientset kubernetes.Interface, nodeName, jobName, namespace, jobImage string, jobCommand []string) error {
	// check if the job already exists
	_, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
	if err == nil {
		log.Info().Msgf("Job %s already exists skipping", jobName)
		return nil
	}
	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "kadmin",
			},
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": nodeName,
					},
					Containers: []corev1.Container{
						{
							Name:    "example-job",
							Image:   jobImage,
							Command: jobCommand,
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Tolerations: []corev1.Toleration{
						{
							Effect:   corev1.TaintEffectNoExecute,
							Operator: corev1.TolerationOpExists,
						},
						{
							Effect:   corev1.TaintEffectNoSchedule,
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}

	_, err = clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	return err
}

func WaitForJobCompletion(clientset kubernetes.Interface, jobName, namespace string) error {
	for {
		// Get the job object
		job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get job %s: %w", jobName, err)
		}
		log.Info().Msgf("Job %s status: %s", jobName, job.Status.String())
		// Check job conditions for completion or failure
		if job.Status.Succeeded > 0 {
			log.Info().Msgf("Job %s completed successfully", jobName)
			return nil
		}
		if job.Status.Failed > 0 {
			return fmt.Errorf("job %s failed", jobName)
		}
		log.Info().Msgf("Job %s not yet complete, waiting...", jobName)
		// Sleep before checking again
		time.Sleep(5 * time.Second)
	}
}

package cmd

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"

	"github.com/jdambly/kadmin/pkg/client"
	"github.com/jdambly/kadmin/pkg/drain"
	"github.com/jdambly/kadmin/pkg/job"
	kNode "github.com/jdambly/kadmin/pkg/node"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	namespace  string
	jobImage   string
	jobCommand []string
)

// rootCmd defines the base command
var rootCmd = &cobra.Command{
	Use:   "kadmin",
	Short: "Administer Kubernetes clusters with ease",
	Long: `Kadmin is a Kubernetes administration tool to manage nodes and perform 
actions like draining, uncordoning, rebooting, and more in a controlled manner.`,
	Run: runKadmin,
}

// Execute initializes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to create jobs in")
	rootCmd.Flags().StringVar(&jobImage, "job-image", "busybox:latest", "Image to use for the job")
	rootCmd.Flags().StringSliceVar(&jobCommand, "job-command", []string{"sh", "-c", "echo Hello && sleep 5"}, "Command to run in the job")
}

func runKadmin(cmd *cobra.Command, args []string) {
	clientset, err := client.NewKubeClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Kubernetes client")
	}

	nodes, err := clientset.CoreV1().Nodes().List(cmd.Context(), metav1.ListOptions{})
	if err != nil {
		log.Fatal().Err(err).Msg("Error retrieving nodes")
	}

	for _, node := range nodes.Items {
		// todo: add a flag to skip master nodes
		// todo: add a flag to skip nodes with a specific label
		// todo: add a flag to skip nodes with a specific annotation
		// todo: add a flag to skip nodes with a specific taint
		// todo: check to see if the node has already been processed
		// todo: check to see if all the nodes are ready and report an error to the user if they are not
		// todo: if there is already a job running on the node, skip it
		// todo: if there is a completed job on the node, skip it
		// todo: add that etcd is running on the master before going to the next ones
		// todo: validate the state of ceph before doing anything
		// todo: add checks to make sure OSDs are up before doing anything
		// todo: add some type of delta logic and compare the states before and after, this would be ceph and etcd
		nodeName := node.Name
		log.Info().Str("node", nodeName).Msg("Processing node")

		if err := drain.DrainNode(clientset, nodeName); err != nil {
			log.Error().Err(err).Str("node", nodeName).Msg("Failed to drain node")
			continue
		}

		// if the job cannot be create exit the program
		// Todo: add a flag to continue processing nodes even if there is an error
		jobName := "job-on-" + nodeName
		if err := job.CreateNodeJob(clientset, nodeName, jobName, namespace, jobImage, jobCommand); err != nil {
			log.Error().Err(err).Str("node", nodeName).Msg("Failed to create job for node")
			if err := drain.UncordonNode(clientset, nodeName); err != nil {
				log.Error().Err(err).Str("node", nodeName).Msg("Failed to uncordon node")
				os.Exit(1)
			}
			os.Exit(1)
		}

		if err := job.WaitForJobCompletion(clientset, jobName, namespace); err != nil {
			log.Error().Err(err).Str("node", nodeName).Msg("Failed to wait for job completion")
			continue
		}

		if err := drain.UncordonNode(clientset, nodeName); err != nil {
			log.Error().Err(err).Str("node", nodeName).Msg("Failed to uncordon node")
			continue
		}

		if err := kNode.WaitForPodsReady(clientset, nodeName); err != nil {
			log.Error().Err(err).Str("node", nodeName).Msg("Failed to wait for pods to become ready")
			continue
		}

		if err := kNode.AnnotateNode(clientset, nodeName, "job-complete", time.Now().Format(time.RFC3339)); err != nil {
			log.Error().Err(err).Str("node", nodeName).Msg("Failed to annotate node")
			continue
		}

		log.Info().Str("node", nodeName).Msg("Successfully completed processing")
	}
}

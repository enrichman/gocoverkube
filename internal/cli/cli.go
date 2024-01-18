package cli

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	gcmd "github.com/enrichman/gocoverkube/internal/cmd"
)

func NewRootCmd(clientset kubernetes.Interface, config *rest.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gocoverkube",
		Short: "gocoverkube",
	}

	rootCmd.AddCommand(
		NewInitCmd(clientset),
		NewCollectCmd(clientset, config),
		NewClearCmd(clientset),
	)

	return rootCmd
}

func NewInitCmd(clientset kubernetes.Interface) *cobra.Command {
	var namespace string

	initCmd := &cobra.Command{
		Use:     "init",
		Short:   "init",
		Args:    cobra.ExactArgs(1),
		PreRunE: CheckConnectionPreRun(clientset),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentName := args[0]

			return gcmd.Init(cmd.Context(), clientset, namespace, deploymentName)
		},
	}

	initCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace")

	return initCmd
}

func NewCollectCmd(clientset kubernetes.Interface, config *rest.Config) *cobra.Command {
	var namespace string

	initCmd := &cobra.Command{
		Use:     "collect",
		Short:   "collect",
		Args:    cobra.ExactArgs(1),
		PreRunE: CheckConnectionPreRun(clientset),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentName := args[0]

			return gcmd.Collect(cmd.Context(), clientset, config, namespace, deploymentName)
		},
	}

	initCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace")

	return initCmd
}

func NewClearCmd(clientset kubernetes.Interface) *cobra.Command {
	var namespace string

	initCmd := &cobra.Command{
		Use:     "clear",
		Short:   "clear",
		Args:    cobra.ExactArgs(1),
		PreRunE: CheckConnectionPreRun(clientset),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentName := args[0]

			return gcmd.Clear(cmd.Context(), clientset, namespace, deploymentName)
		},
	}

	initCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace")

	return initCmd
}

func CheckConnectionPreRun(clientset kubernetes.Interface) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		_, err := gcmd.ServerVersion(clientset)
		return err
	}
}

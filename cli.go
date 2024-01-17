package main

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

func NewRootCmd(clientset kubernetes.Interface) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gocoverkube",
		Short: "gocoverkube",
	}

	rootCmd.AddCommand(
		NewInitCmd(clientset),
	)

	return rootCmd
}

func NewInitCmd(clientset kubernetes.Interface) *cobra.Command {
	var namespace string

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "init",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkConnection(clientset); err != nil {
				return err
			}

			deploymentName := args[0]
			return Init(cmd.Context(), clientset, namespace, deploymentName)
		},
	}

	initCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace")

	return initCmd
}

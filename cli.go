package main

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	openapiclient "k8s.io/client-go/openapi"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/cmd/cp"
	"k8s.io/kubectl/pkg/util/openapi"
	"k8s.io/kubectl/pkg/validation"
)

func NewRootCmd(clientset kubernetes.Interface) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gocoverkube",
		Short: "gocoverkube",
	}

	rootCmd.AddCommand(
		NewInitCmd(clientset),
		NewCollectCmd(clientset),
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

func NewCollectCmd(clientset kubernetes.Interface) *cobra.Command {
	var namespace string

	initCmd := &cobra.Command{
		Use:   "collect",
		Short: "collect",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := checkConnection(clientset)
			if err != nil {
				return err
			}

			o := cp.NewCopyOptions(genericiooptions.IOStreams{})
			err = o.Complete(&factoryImpl{}, cmd, []string{})
			if err != nil {
				return err
			}

			deploymentName := args[0]
			return Collect(cmd.Context(), clientset, namespace, deploymentName)
		},
	}

	initCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace")

	return initCmd
}

type factoryImpl struct {
	namespace  string
	clientSet  *kubernetes.Clientset
	restConfig *rest.Config
}

type clientConfig struct {
	namespace string
}

func (c *clientConfig) RawConfig() (clientcmdapi.Config, error) { return clientcmdapi.Config{}, nil }
func (c *clientConfig) ClientConfig() (*rest.Config, error)     { return nil, nil }
func (c *clientConfig) Namespace() (string, bool, error)        { return c.namespace, false, nil }
func (c *clientConfig) ConfigAccess() clientcmd.ConfigAccess    { return nil }

func (f *factoryImpl) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return &clientConfig{f.namespace}
}

func (f *factoryImpl) KubernetesClientSet() (*kubernetes.Clientset, error) {
	return f.clientSet, nil
}

func (f *factoryImpl) ToRESTConfig() (*rest.Config, error) {
	return f.restConfig, nil
}

//////

func (f *factoryImpl) ToRESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}

func (f *factoryImpl) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return nil, nil
}

func (f *factoryImpl) DynamicClient() (dynamic.Interface, error) {
	return nil, nil
}

func (f *factoryImpl) RESTClient() (*rest.RESTClient, error) {
	return nil, nil
}

func (f *factoryImpl) NewBuilder() *resource.Builder {
	return nil
}

func (f *factoryImpl) ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	return nil, nil
}

func (f *factoryImpl) UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	return nil, nil
}

func (f *factoryImpl) Validator(validationDirective string) (validation.Schema, error) {
	return nil, nil
}

func (f *factoryImpl) OpenAPISchema() (openapi.Resources, error) {
	return nil, nil
}

func (f *factoryImpl) OpenAPIV3Client() (openapiclient.Client, error) {
	return nil, nil
}

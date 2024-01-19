package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	gcmd "github.com/enrichman/gocoverkube/internal/cmd"
)

var Version = "0.0.0-dev"

type RootCfg struct {
	kubeconfig string
	namespace  string

	client *kubernetes.Clientset
	config *rest.Config
}

func NewRootCmd() *cobra.Command {
	rootCfg := &RootCfg{
		kubeconfig: filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		namespace:  v1.NamespaceDefault,
	}

	rootCmd := &cobra.Command{
		Use:   "gocoverkube",
		Short: "gocoverkube",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := initializeConfig(cmd)
			if err != nil {
				return err
			}

			clientset, config, err := newKubernetesClient(rootCfg.kubeconfig)
			if err != nil {
				return err
			}

			rootCfg.client = clientset
			rootCfg.config = config
			return nil
		},
	}

	rootCmd.AddCommand(
		NewInitCmd(rootCfg),
		NewCollectCmd(rootCfg),
		NewClearCmd(rootCfg),
		NewVersionCmd(),
	)

	rootCmd.PersistentFlags().StringVar(&rootCfg.kubeconfig, "kubeconfig", rootCfg.kubeconfig, "kubeconfig [KUBECONFIG]")
	rootCmd.PersistentFlags().StringVarP(&rootCfg.namespace, "namespace", "n", rootCfg.namespace, "namespace [NAMESPACE]")

	return rootCmd
}

func NewInitCmd(rootCfg *RootCfg) *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "init",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		PreRunE:      CheckConnectionPreRun(rootCfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentName := args[0]

			return gcmd.Init(cmd.Context(), rootCfg.client, rootCfg.namespace, deploymentName)
		},
	}
}

func NewCollectCmd(rootCfg *RootCfg) *cobra.Command {
	return &cobra.Command{
		Use:          "collect",
		Short:        "collect",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		PreRunE:      CheckConnectionPreRun(rootCfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentName := args[0]

			return gcmd.Collect(cmd.Context(), rootCfg.client, rootCfg.config, rootCfg.namespace, deploymentName)
		},
	}
}

func NewClearCmd(rootCfg *RootCfg) *cobra.Command {
	return &cobra.Command{
		Use:          "clear",
		Short:        "clear",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		PreRunE:      CheckConnectionPreRun(rootCfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			deploymentName := args[0]

			return gcmd.Clear(cmd.Context(), rootCfg.client, rootCfg.namespace, deploymentName)
		},
	}
}

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}
}

func CheckConnectionPreRun(rootCfg *RootCfg) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		_, err := gcmd.ServerVersion(rootCfg.client)
		return err
	}
}

func newKubernetesClient(kubeconfig string) (*kubernetes.Clientset, *rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, config, nil
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	return bindFlags(cmd, v)
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) error {
	var err error

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			setErr := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if setErr != nil && err == nil {
				err = setErr
			}
		}
	})

	return err
}

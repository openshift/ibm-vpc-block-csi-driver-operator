package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	k8sflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/operator"
	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/version"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	pflag.CommandLine.SetNormalizeFunc(k8sflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	command := NewOperatorCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func NewOperatorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibm-vpc-block-csi-driver-operator",
		Short: "OpenShift IBM VPC Block CSI Driver Operator",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	ctrlCmd := controllercmd.NewControllerCommandConfig(
		"ibm-vpc-block-csi-driver-operator",
		version.Get(),
		operator.RunOperator,
	).NewCommandWithContext(context.Background())
	ctrlCmd.Use = "start"
	ctrlCmd.Short = "Start the IBM VPC Block CSI Driver Operator"

	cmd.AddCommand(ctrlCmd)

	return cmd
}

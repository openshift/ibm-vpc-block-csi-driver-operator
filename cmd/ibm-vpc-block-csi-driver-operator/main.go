package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"k8s.io/component-base/cli"

	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/operator"
	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/version"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
)

func main() {
	command := NewOperatorCommand()
	code := cli.Run(command)
	os.Exit(code)
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

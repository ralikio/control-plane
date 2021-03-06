package command

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/kyma-project/control-plane/tools/cli/pkg/printer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RuntimeCommand represents an execution of the kcp runtimes command
type RuntimeCommand struct {
	cobraCmd *cobra.Command
	log      logger.Logger
	output   string
	params   runtime.ListParameters
}

const (
	inProgress = "in progress"
	succeeded  = "succeeded"
	failed     = "failed"
)

type operationType string

const (
	provision    operationType = "provision"
	deprovision  operationType = "deprovision"
	upgradeKyma  operationType = "kyma upgrade"
	suspension   operationType = "suspension"
	unsuspension operationType = "unsuspension"
)

var tableColumns = []printer.Column{
	{
		Header:    "GLOBALACCOUNT ID",
		FieldSpec: "{.GlobalAccountID}",
	},
	{
		Header:    "SUBACCOUNT ID",
		FieldSpec: "{.SubAccountID}",
	},
	{
		Header:    "SHOOT",
		FieldSpec: "{.ShootName}",
	},
	{
		Header:    "REGION",
		FieldSpec: "{.ProviderRegion}",
	},
	{
		Header:    "PLAN",
		FieldSpec: "{.ServicePlanName}",
	},
	{
		Header:         "CREATED AT",
		FieldFormatter: runtimeCreatedAt,
	},
	{
		Header:         "STATE",
		FieldFormatter: runtimeStatus,
	},
}

// NewRuntimeCmd constructs a new instance of RuntimeCommand and configures it in terms of a cobra.Command
func NewRuntimeCmd() *cobra.Command {
	cmd := RuntimeCommand{}
	cobraCmd := &cobra.Command{
		Use:     "runtimes",
		Aliases: []string{"runtime", "rt"},
		Short:   "Displays Kyma Runtimes.",
		Long: `Displays Kyma Runtimes and their primary attributes, such as identifiers, region, or states.
The command supports filtering Runtimes based on various attributes. See the list of options for more details.`,
		Example: `  kcp runtimes                                           Display table overview about all Runtimes.
  kcp rt -c c-178e034 -o json                            Display all details about one Runtime identified by a Shoot name in the JSON format.
  kcp runtimes --account CA4836781TID000000000123456789  Display all Runtimes of a given global account.`,
		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	SetOutputOpt(cobraCmd, &cmd.output)
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Shoots, "shoot", "c", nil, "Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.GlobalAccountIDs, "account", "g", nil, "Filter by global account ID. You can provide multiple values, either separated by a comma (e.g. GAID1,GAID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.SubAccountIDs, "subaccount", "s", nil, "Filter by subaccount ID. You can provide multiple values, either separated by a comma (e.g. SAID1,SAID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.RuntimeIDs, "runtime-id", "i", nil, "Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Regions, "region", "r", nil, "Filter by provider region. You can provide multiple values, either separated by a comma (e.g. westeurope,northeurope), or by specifying the option multiple times.")
	cobraCmd.Flags().StringSliceVarP(&cmd.params.Plans, "plan", "p", nil, "Filter by service plan name. You can provide multiple values, either separated by a comma (e.g. azure,trial), or by specifying the option multiple times.")

	return cobraCmd
}

// Run executes the runtimes command
func (cmd *RuntimeCommand) Run() error {
	cmd.log = logger.New()
	client := runtime.NewClient(cmd.cobraCmd.Context(), GlobalOpts.KEBAPIURL(), CLICredentialManager(cmd.log))

	rp, err := client.ListRuntimes(cmd.params)
	if err != nil {
		return errors.Wrap(err, "while listing runtimes")
	}
	err = cmd.printRuntimes(rp)
	if err != nil {
		return errors.Wrap(err, "while printing runtimes")
	}

	return nil
}

// Validate checks the input parameters of the runtimes command
func (cmd *RuntimeCommand) Validate() error {
	err := ValidateOutputOpt(cmd.output)
	if err != nil {
		return err
	}
	return nil
}

func (cmd *RuntimeCommand) printRuntimes(runtimes runtime.RuntimesPage) error {
	switch cmd.output {
	case tableOutput:
		tp, err := printer.NewTablePrinter(tableColumns, false)
		if err != nil {
			return err
		}
		return tp.PrintObj(runtimes.Data)
	case jsonOutput:
		jp := printer.NewJSONPrinter("  ")
		jp.PrintObj(runtimes)
	}

	return nil
}

func runtimeStatus(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return operationStatusToString(findLastOperation(rt))
}

func findLastOperation(rt runtime.RuntimeDTO) (runtime.Operation, operationType) {
	op := *rt.Status.Provisioning
	opType := provision
	// Take the first upgrade operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.UpgradingKyma.Count > 0 {
		op = rt.Status.UpgradingKyma.Data[0]
		opType = upgradeKyma
	}

	// Take the first unsuspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Unsuspension.Count > 0 && rt.Status.Unsuspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Unsuspension.Data[0]
		opType = unsuspension
	}

	// Take the first suspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Suspension.Count > 0 && rt.Status.Suspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Suspension.Data[0]
		opType = suspension
	}

	if rt.Status.Deprovisioning != nil && rt.Status.Deprovisioning.CreatedAt.After(op.CreatedAt) {
		op = *rt.Status.Deprovisioning
		opType = deprovision
	}

	return op, opType
}

func operationStatusToString(op runtime.Operation, t operationType) string {
	switch op.State {
	case succeeded:
		switch t {
		case deprovision:
			return "deprovisioned"
		case suspension:
			return "suspended"
		}
		return "succeeded"
	case failed:
		return fmt.Sprintf("%s (%s)", "failed", t)
	case inProgress:
		switch t {
		case provision:
			return "provisioning"
		case unsuspension:
			return "provisioning (unsuspending)"
		case deprovision:
			return "deprovisioning"
		case suspension:
			return "deprovisioning (suspending)"
		case upgradeKyma:
			return "upgrading"
		}
	}

	return "succeeded"
}

func runtimeCreatedAt(obj interface{}) string {
	rt := obj.(runtime.RuntimeDTO)
	return rt.Status.CreatedAt.Format("2006/01/02 15:04:05")
}

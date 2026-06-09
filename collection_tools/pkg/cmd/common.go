// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/constants"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

var (
	kubeConfig      string
	outputFile      string
	useAnalyserJSON bool
	ptpInterface    string
	nodeName        string
	clockType       string
)

func AddKubeconfigFlag(targetCmd *cobra.Command) {
	targetCmd.Flags().StringVarP(&kubeConfig,
		"kubeconfig",
		"k", "",
		"Path to the kubeconfig file")
	err := targetCmd.MarkFlagRequired("kubeconfig")
	utils.IfErrorExitOrPanic(err)
}

func AddOutputFlag(targetCmd *cobra.Command) {
	targetCmd.Flags().StringVarP(&outputFile,
		"output",
		"o", "",
		"Path to the output file")
}

func AddFormatFlag(targetCmd *cobra.Command) {
	targetCmd.Flags().BoolVarP(
		&useAnalyserJSON,
		"use-analyser-format",
		"j",
		false,
		"Indent JSON output (detect always emits a JSON array)",
	)
}

func AddInterfaceFlag(targetCmd *cobra.Command) {
	targetCmd.Flags().StringVarP(&ptpInterface,
		"interface",
		"i", "",
		"Name of the PTP interface")
	err := targetCmd.MarkFlagRequired("interface")
	utils.IfErrorExitOrPanic(err)
}

func AddNodeNameFlag(targetCmd *cobra.Command) {
	targetCmd.Flags().StringVarP(&nodeName,
		"nodeName",
		"n", "",
		"Name of the Node under test (valid only for MNO Use case)")
}

func AddClockTypeFlag(targetCmd *cobra.Command) {
	targetCmd.Flags().StringVarP(&clockType,
		"clock-type",
		"c", constants.ClockTypeGM,
		"Clock type: GM (Grand Master) or BC (Boundary Clock)")
}

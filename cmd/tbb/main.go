// Copyright 2020 The the-blockchain-bar Authors
// This file is part of the the-blockchain-bar library.
//
// The the-blockchain-bar library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The the-blockchain-bar library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"fmt"
	"os"

	"github.com/IacopoMelani/the-blockchain-pub/fs"
	"github.com/spf13/cobra"
)

const flagKeystoreFile = "keystore"
const flagDataDir = "datadir"
const flagMiner = "miner"
const flagSSLEmail = "ssl-email"
const flagDisableSSL = "disable-ssl"
const flagIP = "ip"
const flagPort = "port"
const flagBootstrapAcc = "bootstrap-account"
const flagBootstrapIp = "bootstrap-ip"
const flagBootstrapPort = "bootstrap-port"
const flagAmount = "amount"
const flagToAddress = "to"
const flagPassword = "pwd"
const flagConfirm = "confirm"

func main() {
	var tbbCmd = &cobra.Command{
		Use:   "tbb",
		Short: "The Blockchain Bar CLI",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	tbbCmd.AddCommand(versionCmd)
	tbbCmd.AddCommand(balancesCmd())
	tbbCmd.AddCommand(walletCmd())
	tbbCmd.AddCommand(runCmd())

	err := tbbCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to your node's data dir where the DB will be/is stored")
	cmd.MarkFlagRequired(flagDataDir)
}

func addKeystoreFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagKeystoreFile, "", "Absolute path to the encrypted keystore file")
	cmd.MarkFlagRequired(flagKeystoreFile)
}

func addAmountFlag(cmd *cobra.Command) {
	cmd.Flags().Uint(flagAmount, 0, "Amount to send")
}

func addToAddressFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagToAddress, "", "Address to send the funds to")
}

func addPwdFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagPassword, "", "Password to unlock the keystore, use with caution")
}

func addConfirmFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(flagConfirm, false, "Confirm the action")
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)

	return fs.ExpandPath(dataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}

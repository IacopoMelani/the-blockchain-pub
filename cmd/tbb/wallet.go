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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/IacopoMelani/the-blockchain-bar/database"
	"github.com/IacopoMelani/the-blockchain-bar/node"
	"github.com/IacopoMelani/the-blockchain-bar/wallet"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
)

func walletCmd() *cobra.Command {
	var walletCmd = &cobra.Command{
		Use:   "wallet",
		Short: "Manages blockchain accounts and keys.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return incorrectUsageErr()
		},
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	walletCmd.AddCommand(walletNewAccountCmd())
	walletCmd.AddCommand(walletPrintPrivKeyCmd())
	walletCmd.AddCommand(walletSendTransaction())

	return walletCmd
}

func walletNewAccountCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "new-account",
		Short: "Creates a new account with a new set of a elliptic-curve Private + Public keys.",
		Run: func(cmd *cobra.Command, args []string) {
			password := getPassPhrase("Please enter a password to encrypt the new wallet:", true)
			dataDir := getDataDirFromCmd(cmd)

			acc, err := wallet.NewKeystoreAccount(dataDir, password)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Printf("New account created: %s\n", acc.Hex())
			fmt.Printf("Saved in: %s\n", wallet.GetKeystoreDirPath(dataDir))
		},
	}

	addDefaultRequiredFlags(cmd)

	return cmd
}

func walletPrintPrivKeyCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "pk-print",
		Short: "Unlocks keystore file and prints the Private + Public keys.",
		Run: func(cmd *cobra.Command, args []string) {
			ksFile, _ := cmd.Flags().GetString(flagKeystoreFile)
			password := getPassPhrase("Please enter a password to decrypt the wallet:", false)

			keyJson, err := ioutil.ReadFile(ksFile)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			key, err := keystore.DecryptKey(keyJson, password)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			spew.Dump(key)
		},
	}

	addKeystoreFlag(cmd)

	return cmd
}

func walletSendTransaction() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "send-transaction",
		Short: "Sends a transaction to the blockchain.",
		Run: func(cmd *cobra.Command, args []string) {
			ksFile, _ := cmd.Flags().GetString(flagKeystoreFile)

			password, _ := cmd.Flags().GetString(flagPassword)
			if password == "" {
				password = getPassPhrase("Please enter a password to decrypt the wallet:", false)
			}

			keyJson, err := ioutil.ReadFile(ksFile)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			key, err := keystore.DecryptKey(keyJson, password)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			amount, _ := cmd.Flags().GetUint(flagAmount)

			if amount == 0 {
				amount, err = GetTransactionAmount()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
			}

			toAddress, _ := cmd.Flags().GetString(flagToAddress)

			if toAddress == "" {
				toAddress, err = GetToAddress()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
			}

			nextNonceRawBody, err := makeRequest("http://localhost:8111/node/nonce/next", "POST", map[string]interface{}{
				"account": key.Address.Hex(),
			})
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			var nextNonceRes node.NextNonceRes
			err = json.Unmarshal(nextNonceRawBody, &nextNonceRes)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			tx := database.NewTx(key.Address, database.NewAccount(toAddress), amount, nextNonceRes.Nonce, "")

			signedTx, err := wallet.SignTxWithKeystoreAccount(tx, key.Address, password, filepath.Dir(ksFile))
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			txHash, err := signedTx.Hash()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			fmt.Printf("\nSending transaction: %s 🚀🚀🚀\n", txHash.Hex())
			fmt.Printf("\tAmount: '%v'\n", signedTx.Value)
			fmt.Printf("\tTo: '%v'\n", signedTx.To.Hex())
			fmt.Printf("\tFees: '%v'\n", database.TxFee)

			if confirm, _ := cmd.Flags().GetBool(flagConfirm); !confirm {

				fmt.Println("\n\nConfirm transaction? (y/n)")
				var confirm string
				fmt.Scanln(&confirm)

				if confirm != "y" {
					fmt.Println("\n\nAborting transaction...")
					os.Exit(0)
				}
			}

			rawBytes, err := rlp.EncodeToBytes(signedTx)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			rawTx := hexutil.Encode(rawBytes)

			fmt.Printf("Sending transaction to the blockchain...\n")

			body, err := makeRequest("http://localhost:8111/tx/add", "POST", map[string]interface{}{
				"tx": rawTx,
			})
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			fmt.Printf("%s\n", body)
		},
	}

	addKeystoreFlag(cmd)
	addToAddressFlag(cmd)
	addAmountFlag(cmd)
	addPwdFlag(cmd)
	addConfirmFlag(cmd)

	return cmd
}

func getPassPhrase(prompt string, confirmation bool) string {
	return utils.GetPassPhrase(prompt, confirmation)
}

// ask to user to insert amount of transaction
func GetTransactionAmount() (uint, error) {
	var value uint
	fmt.Print("Insert transaction amount: ")
	fmt.Scanln(&value)
	if value == 0 {
		return 0, fmt.Errorf("invalid amount")
	}
	return value, nil
}

func GetToAddress() (string, error) {
	var toAddress string
	fmt.Print("Insert to address: ")
	fmt.Scanln(&toAddress)
	if toAddress == "" {
		return "", fmt.Errorf("invalid to address")
	}
	return toAddress, nil
}

// make POST request with JSON body
func makeRequest(url string, method string, data map[string]interface{}) ([]byte, error) {

	if data == nil {
		data = make(map[string]interface{})
	}

	jsonBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

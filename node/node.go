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
package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/IacopoMelani/the-blockchain-pub/database"
)

const DefaultBootstrapIp = "node.tbb.web3.coach"

// The Web3Coach's Genesis account with 1M TBB tokens
const DefaultBootstrapAcc = "0x50543e830590fD03a0301fAA0164d731f0E2ff7D"
const DefaultMiner = "0x0000000000000000000000000000000000000000"
const DefaultIP = "127.0.0.1"
const HttpSSLPort = 443

const endpointBalancesList = "/balances/list"

const endpointTxAdd = "/tx/add"

const endpointStatus = "/node/status"

const endpointSync = "/node/sync"
const endpointSyncQueryKeyFromBlock = "fromBlock"
const endpointSyncQueryKeyMode = "mode"
const endpointSyncQueryKeyLast = "last"
const endpointSyncQueryKeyModeAfter = "after"
const endpointSyncQueryKeyModeBefore = "before"

const endpointAddPeer = "/node/peer"
const endpointAddPeerQueryKeyIP = "ip"
const endpointAddPeerQueryKeyPort = "port"
const endpointAddPeerQueryKeyMiner = "miner"
const endpointAddPeerQueryKeyVersion = "version"

const endpointNextNonce = "/address/nonce/next"

const endtpointAddressBalance = "/address/balance"

const endpointAddressTransactions = "/address/transactions"

const miningIntervalSeconds = 3
const DefaultMiningDifficulty = 2

type PeerNode struct {
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Account     common.Address `json:"account"`
	NodeVersion string         `json:"node_version"`

	// Whenever my node already established connection, sync with this Peer
	connected bool
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func (pn PeerNode) ApiProtocol() string {
	if pn.Port == HttpSSLPort {
		return "https"
	}

	return "http"
}

type Node struct {
	dataDir string
	info    PeerNode

	// The main blockchain state after all TXs from mined blocks were applied
	state *database.State

	// temporary pending state validating new incoming TXs but reset after the block is mined
	pendingState *database.State

	knownPeers      map[string]PeerNode
	pendingTXs      map[string]database.SignedTx
	archivedTXs     map[string]database.SignedTx
	newSyncedBlocks chan database.Block
	newPendingTXs   chan database.SignedTx
	nodeVersion     string

	// Number of zeroes the hash must start with to be considered valid. Default 3
	miningDifficulty uint64
	isMining         bool
}

func New(dataDir string, ip string, port uint64, acc common.Address, bootstrap PeerNode, version string, miningDifficulty uint64) *Node {
	knownPeers := make(map[string]PeerNode)

	n := &Node{
		dataDir:          dataDir,
		info:             NewPeerNode(ip, port, false, acc, true, version),
		knownPeers:       knownPeers,
		pendingTXs:       make(map[string]database.SignedTx),
		archivedTXs:      make(map[string]database.SignedTx),
		newSyncedBlocks:  make(chan database.Block),
		newPendingTXs:    make(chan database.SignedTx, 10000),
		nodeVersion:      version,
		isMining:         false,
		miningDifficulty: miningDifficulty,
	}

	n.AddPeer(bootstrap)

	return n
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, acc common.Address, connected bool, version string) PeerNode {
	return PeerNode{ip, port, isBootstrap, acc, version, connected}
}

func (n *Node) Run(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	fmt.Printf("Listening on: %s:%d\n", n.info.IP, n.info.Port)

	state, err := database.NewStateFromDisk(n.dataDir, n.miningDifficulty)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state

	pendingState := state.Copy()
	n.pendingState = &pendingState

	if err = func() error {

		if err := n.CheckDifficulty(); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	fmt.Println("Blockchain state:")
	fmt.Printf("	- height: %d\n", n.state.LatestBlock().Header.Number)
	fmt.Printf("	- hash: %s\n", n.state.LatestBlockHash().Hex())
	fmt.Printf("	- difficulty: %d\n", n.state.LatestBlock().Header.Difficulty)

	go n.sync(ctx)
	go n.mine(ctx)

	return n.serveHttp(ctx, isSSLDisabled, sslEmail)
}

func (n *Node) LatestBlockHash() database.Hash {
	return n.state.LatestBlockHash()
}

func (n *Node) serveHttp(ctx context.Context, isSSLDisabled bool, sslEmail string) error {

	e := echo.New()

	e.Use(middleware.Recover())

	e.GET(endpointBalancesList, func(c echo.Context) error {
		return listBalancesHandler(c, n)
	})

	e.POST(endtpointAddressBalance, func(c echo.Context) error {
		return addressBalanceHandler(c, n)
	})

	e.POST(endpointTxAdd, func(c echo.Context) error {
		return txAddHandler(c, n)
	})

	e.GET(endpointStatus, func(c echo.Context) error {
		return statusHandler(c, n)
	})

	e.GET(endpointSync, func(c echo.Context) error {
		return syncHandler(c, n)
	})

	e.GET(endpointAddPeer, func(c echo.Context) error {
		return addPeerHandler(c, n)
	})

	e.POST(endpointNextNonce, func(c echo.Context) error {
		return nextNonceHandler(c, n)
	})

	e.POST(endpointAddressTransactions, func(c echo.Context) error {
		return transactionsHandler(c, n)
	})

	if isSSLDisabled {
		server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port), Handler: e}

		go func() {
			<-ctx.Done()
			_ = server.Close()
		}()

		err := server.ListenAndServe()
		// This shouldn't be an error!
		if err != http.ErrServerClosed {
			panic(err)
		}

		return nil

	} else {

		certmagic.DefaultACME.Email = sslEmail

		return certmagic.HTTPS([]string{n.info.IP}, e)
	}
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {

				if !n.IsMining() {
					n.setMining(true)

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
					}

					n.setMining(false)
				}
			}()

		case block := <-n.newSyncedBlocks:
			if n.IsMining() {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", blockHash.Hex())

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {

	difficulty := n.miningDifficulty

	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Account,
		difficulty,
		n.getPendingTXsAsArray(),
	)

	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)

	err = n.addBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block database.Block) {

	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory Pending TXs Pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

func (n *Node) setMining(value bool) {
	n.isMining = value
}

func (n *Node) CheckDifficulty() error {

	if n.state.LatestBlock().Header.Number%uint64(database.BlockNumberToCheckDifficulty) == 0 {
		difficulty, err := n.GetNewDifficulty()
		if err != nil {
			return err
		}
		n.ChangeMiningDifficulty(difficulty)
	}

	return nil
}

func (n *Node) GetAproximateBlockResolutionTime() (time.Duration, error) {

	blocks, err := database.GetBlocksBefore(n.LatestBlockHash(), database.BlockNumberToCheckDifficulty, n.dataDir)
	if err != nil {
		return 0, err
	}

	// diff time beetween first and last block mined in blocks slice

	if len(blocks) == 0 {
		return 0, nil
	}

	firstBlock := blocks[0]
	lastBlock := blocks[len(blocks)-1]

	firstBlockTime := time.Unix(int64(firstBlock.Value.Header.Time), 0)
	lastBlockTime := time.Unix(int64(lastBlock.Value.Header.Time), 0)

	diff := lastBlockTime.Sub(firstBlockTime)

	return diff / time.Duration(len(blocks)), nil
}

func (n *Node) GetNewDifficulty() (uint64, error) {

	if n.miningDifficulty == 0 {
		return 0, errors.New("mining difficulty is 0")
	}

	average, err := n.GetAproximateBlockResolutionTime()
	if err != nil {
		return 0, err
	}

	if average == 0 {
		return n.miningDifficulty, nil
	}

	if average < database.MiningAproxTime {
		return (n.miningDifficulty + 1), nil
	} else if average > database.MiningAproxTime {
		return (n.miningDifficulty - 1), nil
	} else {
		return n.miningDifficulty, nil
	}
}

func (n *Node) ChangeMiningDifficulty(newDifficulty uint64) {
	n.miningDifficulty = newDifficulty
	n.state.ChangeMiningDifficulty(newDifficulty)
	fmt.Printf("Change mining difficulty to: %d\n", newDifficulty)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}

func (n *Node) IsMining() bool {
	return n.isMining
}

func (n *Node) AddPendingTX(tx database.SignedTx, fromPeer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	for _, pendingTx := range n.pendingTXs {
		if tx.From.Hash() == pendingTx.From.Hash() && pendingTx.Nonce == tx.Nonce {
			return fmt.Errorf("TX with same nonce already pending")
		}
	}

	if !isAlreadyPending && !isArchived {

		err = n.validateTxBeforeAddingToMempool(tx)
		if err != nil {
			return err
		}

		fmt.Printf("Added Pending TX %s from Peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
	}

	return nil
}

// addBlock is a wrapper around the n.state.AddBlock() to have a single function for changing the main state
// from the Node perspective, so we can also reset the pending state in the same time.
func (n *Node) addBlock(block database.Block) error {

	defer func() {
		// Reset the pending state
		pendingState := n.state.Copy()
		n.pendingState = &pendingState
	}()

	_, err := n.state.AddBlock(block)
	if err != nil {
		return err
	}

	if err := n.CheckDifficulty(); err != nil {
		fmt.Printf("Error checking difficulty: %s\n", err)
	}

	return nil
}

// resetChain is a wrapper around the n.state.ResetChain() to have a single function for changing the main state
func (n *Node) resetChain() error {

	n.state.ResetChain(n.dataDir)
	pendingState := n.state.Copy()
	n.pendingState = &pendingState

	return nil
}

// validateTxBeforeAddingToMempool ensures the TX is authentic, with correct nonce, and the sender has sufficient
// funds so we waste PoW resources on TX we can tell in advance are wrong.
func (n *Node) validateTxBeforeAddingToMempool(tx database.SignedTx) error {
	return database.ApplyTx(tx, n.pendingState)
}

func (n *Node) getPendingTXsAsArray() []database.SignedTx {
	txs := make([]database.SignedTx, len(n.pendingTXs))

	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Nonce < txs[j].Nonce
	})

	return txs
}

func (n *Node) GetPendingTXsExtendedAsArrayByAccount(acc common.Address) ([]database.SignedTxExtended, error) {
	txs := make([]database.SignedTxExtended, 0)

	for _, tx := range n.pendingTXs {
		if tx.From == acc {
			txExtended, err := database.NewSignedTxExtended(tx, database.Block{})
			if err != nil {
				return nil, err
			}
			txs = append(txs, txExtended)
		}
	}

	return txs, nil
}

func (n *Node) GetTxsByAccountAndType(account common.Address, txType string, last int) ([]database.SignedTxExtended, error) {

	txs := make([]database.SignedTxExtended, 0)
	count := 0

	blocks, err := database.GetBlocksBefore(n.LatestBlockHash(), int64(n.state.LatestBlock().Header.Number), n.dataDir)
	if err != nil {
		return nil, err
	}

loopBlocks:
	for _, block := range blocks {

		for _, tx := range block.Value.TXs {

			switch txType {
			case "in":
				if tx.To == account {
					txExtended, err := database.NewSignedTxExtended(tx, block.Value)
					if err != nil {
						return nil, err
					}
					txs = append(txs, txExtended)
					count++

				}
			case "out":
				if tx.From == account {
					txExtended, err := database.NewSignedTxExtended(tx, block.Value)
					if err != nil {
						return nil, err
					}
					txs = append(txs, txExtended)
					count++
				}
			}

			if last > 0 && count >= last {
				break loopBlocks
			}
		}
	}

	return txs, nil
}

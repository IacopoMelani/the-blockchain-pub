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
	"fmt"
	"net/http"
	"strconv"

	"github.com/IacopoMelani/the-blockchain-bar/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

type ErrRes struct {
	Error string `json:"error"`
}

type BalancesRes struct {
	Hash     database.Hash           `json:"block_hash"`
	Balances map[common.Address]uint `json:"balances"`
}

type TxAddReq struct {
	RawTx string `json:"tx"`
}

type TxAddRes struct {
	Success bool `json:"success"`
}

type StatusRes struct {
	Hash        database.Hash       `json:"block_hash"`
	Number      uint64              `json:"block_number"`
	KnownPeers  map[string]PeerNode `json:"peers_known"`
	PendingTXs  []database.SignedTx `json:"pending_txs"`
	NodeVersion string              `json:"node_version"`
	Account     common.Address      `json:"account"`
}

type SyncRes struct {
	Blocks []database.Block `json:"blocks"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func listBalancesHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	enableCors(&w)

	writeRes(w, BalancesRes{state.LatestBlockHash(), state.Balances})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := TxAddReq{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	var signedTx database.SignedTx

	rawBytes, err := hexutil.Decode(req.RawTx)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	if err := rlp.DecodeBytes(rawBytes, &signedTx); err != nil {
		writeErrRes(w, err)
		return
	}

	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, TxAddRes{Success: true})
}

func statusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	enableCors(&w)

	res := StatusRes{
		Hash:        node.state.LatestBlockHash(),
		Number:      node.state.LatestBlock().Header.Number,
		KnownPeers:  node.knownPeers,
		PendingTXs:  node.getPendingTXsAsArray(),
		NodeVersion: node.nodeVersion,
		Account:     database.NewAccount(node.info.Account.String()),
	}

	writeRes(w, res)
}

func syncHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	reqHash := r.URL.Query().Get(endpointSyncQueryKeyFromBlock)
	reqMode := r.URL.Query().Get(endpointSyncQueryKeyMode)
	reqLast := r.URL.Query().Get(endpointSyncQueryKeyLast)

	// convert reqLast to int64
	last, err := strconv.ParseInt(reqLast, 10, 64)
	if err != nil {
		last = 10
	}

	if reqMode == "" || (reqMode != endpointSyncQueryKeyModeAfter && reqMode != endpointSyncQueryKeyModeBefore) {
		reqMode = endpointSyncQueryKeyModeBefore
	}

	hash := database.Hash{}
	err = hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrRes(w, err)
		return
	}

	var blocks []database.BlockFS

	switch reqMode {
	case endpointSyncQueryKeyModeAfter:
		blocks, err = database.GetBlocksAfter(hash, last, node.dataDir)
		if err != nil {
			writeErrRes(w, err)
			return
		}
	case endpointSyncQueryKeyModeBefore:
		blocks, err = database.GetBlocksBefore(hash, last, node.dataDir)
		if err != nil {
			writeErrRes(w, err)
			return
		}
	}

	writeRes(w, map[string][]database.BlockFS{"blocks": blocks})
}

func addPeerHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	peerIP := r.URL.Query().Get(endpointAddPeerQueryKeyIP)
	peerPortRaw := r.URL.Query().Get(endpointAddPeerQueryKeyPort)
	minerRaw := r.URL.Query().Get(endpointAddPeerQueryKeyMiner)
	versionRaw := r.URL.Query().Get(endpointAddPeerQueryKeyVersion)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, database.NewAccount(minerRaw), true, versionRaw)

	node.AddPeer(peer)

	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TcpAddress())

	writeRes(w, AddPeerRes{true, ""})
}

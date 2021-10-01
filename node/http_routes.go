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

	"github.com/IacopoMelani/the-blockchain-pub/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/labstack/echo/v4"
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

type NextNonceReq struct {
	Account string `json:"account"`
}

type NextNonceRes struct {
	Nonce uint `json:"nonce"`
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
	Blocks []database.BlockFS `json:"blocks"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func listBalancesHandler(c echo.Context, node *Node) error {
	return c.JSON(http.StatusOK, BalancesRes{node.state.LatestBlockHash(), node.state.Balances})
}

func txAddHandler(c echo.Context, node *Node) error {

	var req TxAddReq

	err := readReq(c.Request(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrRes{err.Error()})
	}

	var signedTx database.SignedTx

	rawBytes, err := hexutil.Decode(req.RawTx)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrRes{err.Error()})
	}

	if err := rlp.DecodeBytes(rawBytes, &signedTx); err != nil {
		return c.JSON(http.StatusBadRequest, ErrRes{err.Error()})
	}

	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrRes{err.Error()})
	}

	return c.JSON(http.StatusOK, TxAddRes{Success: true})
}

func nextNonceHandler(c echo.Context, node *Node) error {

	req := NextNonceReq{}
	err := readReq(c.Request(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrRes{err.Error()})
	}

	nonce := node.state.GetNextAccountNonce(database.NewAccount(req.Account))

	return c.JSON(http.StatusOK, NextNonceRes{Nonce: nonce})
}

func statusHandler(c echo.Context, node *Node) error {
	return c.JSON(http.StatusOK, StatusRes{
		Hash:        node.state.LatestBlockHash(),
		Number:      node.state.LatestBlock().Header.Number,
		KnownPeers:  node.knownPeers,
		PendingTXs:  node.getPendingTXsAsArray(),
		NodeVersion: node.nodeVersion,
		Account:     database.NewAccount(node.info.Account.String()),
	})
}

func syncHandler(c echo.Context, node *Node) error {

	reqHash := c.Request().URL.Query().Get(endpointSyncQueryKeyFromBlock)
	reqMode := c.Request().URL.Query().Get(endpointSyncQueryKeyMode)
	reqLast := c.Request().URL.Query().Get(endpointSyncQueryKeyLast)

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
		return c.JSON(http.StatusBadRequest, ErrRes{err.Error()})
	}

	var blocks []database.BlockFS

	switch reqMode {
	case endpointSyncQueryKeyModeAfter:
		blocks, err = database.GetBlocksAfter(hash, last, node.dataDir)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrRes{err.Error()})
		}
	case endpointSyncQueryKeyModeBefore:
		blocks, err = database.GetBlocksBefore(hash, last, node.dataDir)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrRes{err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string][]database.BlockFS{"blocks": blocks})
}

func addPeerHandler(c echo.Context, node *Node) error {

	r := c.Request()

	peerIP := r.URL.Query().Get(endpointAddPeerQueryKeyIP)
	peerPortRaw := r.URL.Query().Get(endpointAddPeerQueryKeyPort)
	minerRaw := r.URL.Query().Get(endpointAddPeerQueryKeyMiner)
	versionRaw := r.URL.Query().Get(endpointAddPeerQueryKeyVersion)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrRes{err.Error()})
	}

	peer := NewPeerNode(peerIP, peerPort, false, database.NewAccount(minerRaw), true, versionRaw)

	node.AddPeer(peer)

	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TcpAddress())

	return c.JSON(http.StatusOK, AddPeerRes{true, ""})
}

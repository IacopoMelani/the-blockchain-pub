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
	"fmt"
	"time"

	"github.com/IacopoMelani/the-blockchain-pub/database"
	"github.com/ethereum/go-ethereum/common"
)

type PendingBlock struct {
	parent     database.Hash
	number     uint64
	time       uint64
	miner      common.Address
	difficulty uint64
	txs        []database.SignedTx
}

func NewPendingBlock(parent database.Hash, number uint64, miner common.Address, difficulty uint64, txs []database.SignedTx) PendingBlock {
	return PendingBlock{parent, number, uint64(time.Now().Unix()), miner, difficulty, txs}
}

func Mine(ctx context.Context, pb PendingBlock) (database.Block, error) {

	start := time.Now()
	attempt := 0
	var block database.Block
	var hash database.Hash
	var nonce uint32

	for !database.IsBlockHashValid(hash, pb.difficulty) {
		select {
		case <-ctx.Done():
			fmt.Println("Mining cancelled!")

			return database.Block{}, fmt.Errorf("mining cancelled. %s", ctx.Err())
		default:
		}

		nonce = uint32(attempt)
		attempt++

		if attempt%1000000 == 0 || attempt == 1 {
			fmt.Printf("Mining %d Pending TXs. Attempt: %d\n", len(pb.txs), attempt)
		}

		block = database.NewBlock(pb.parent, pb.number, nonce, pb.time, pb.miner, pb.difficulty, pb.txs)
		blockHash, err := block.Hash()
		if err != nil {
			return database.Block{}, fmt.Errorf("couldn't mine block. %s", err.Error())
		}

		hash = blockHash
	}

	fmt.Printf("\nMined new Block '%x' using PoW 🎉🎉🎉\n", hash)
	fmt.Printf("\tHeight: '%v'\n", block.Header.Number)
	fmt.Printf("\tNonce: '%v'\n", block.Header.Nonce)
	fmt.Printf("\tDifficulty: '%v'\n", block.Header.Difficulty)
	fmt.Printf("\tCreated: '%v'\n", block.Header.Time)
	fmt.Printf("\tMiner: '%v'\n", block.Header.Miner.String())
	fmt.Printf("\tParent: '%v'\n\n", block.Header.Parent.Hex())

	fmt.Printf("\tAttempt: '%v'\n", attempt)
	fmt.Printf("\tTime: %s\n\n", time.Since(start))

	return block, nil
}

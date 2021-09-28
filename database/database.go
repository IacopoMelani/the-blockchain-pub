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
package database

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
)

func GetBlocksAfter(blockHash Hash, last int64, dataDir string) ([]BlockFS, error) {
	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}

	blocks := make([]BlockFS, 0)
	shouldStartCollecting := false

	if reflect.DeepEqual(blockHash, Hash{}) {
		shouldStartCollecting = true
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFs BlockFS
		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		if shouldStartCollecting {
			// append the block to the list
			blocks = append(blocks, blockFs)

			if len(blocks) == int(last) {
				break
			}

			continue
		}

		if blockHash == blockFs.Key {
			shouldStartCollecting = true
		}
	}

	return blocks, nil
}

func GetBlocksBefore(blockHash Hash, last int64, dataDir string) ([]BlockFS, error) {

	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}

	blocks := make([]BlockFS, 0)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFs BlockFS
		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, blockFs)

		if blockHash == blockFs.Key {
			break
		}
	}

	// reverse the list
	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	if len(blocks) <= int(last) {
		return blocks, nil
	}

	return blocks[0:last], nil
}

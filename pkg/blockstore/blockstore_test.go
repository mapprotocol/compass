// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package blockstore

import (
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/msg"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "blockstore")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	chain := msg.ChainId(10)

	bs, err := NewBlockstore(dir, chain, constant.ZeroAddress.String(), mapprotocol.RoleOfMaintainer)
	if err != nil {
		t.Fatal(err)
	}
	// Load non-existent dir/file
	block, err := bs.TryLoadLatestBlock()
	if err != nil {
		t.Fatal(err)
	}

	if block.Uint64() != uint64(0) {
		t.Fatalf("Expected: %d got: %d", 0, block.Uint64())
	}

	// Save block number
	block = big.NewInt(999)
	err = bs.StoreBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	// Load block number
	latest, err := bs.TryLoadLatestBlock()
	if err != nil {
		t.Fatal(err)
	}

	if block.Uint64() != latest.Uint64() {
		t.Fatalf("Expected: %d got: %d", block.Uint64(), latest.Uint64())
	}

	// Save block number again
	block = big.NewInt(1234)
	err = bs.StoreBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	// Load block number
	latest, err = bs.TryLoadLatestBlock()
	if err != nil {
		t.Fatal(err)
	}

	if block.Uint64() != latest.Uint64() {
		t.Fatalf("Expected: %d got: %d", block.Uint64(), latest.Uint64())
	}
}

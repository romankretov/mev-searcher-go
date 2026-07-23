package cache

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"searcher-bot/contracts/erc20"
	"sync"
)

type DecimalsCache struct {
	mu       sync.RWMutex
	decimals map[common.Address]uint8
}

func NewDecimalsCache() *DecimalsCache {
	return &DecimalsCache{
		decimals: make(map[common.Address]uint8),
	}
}

func (d *DecimalsCache) Get(address common.Address) (uint8, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	decimal, ok := d.decimals[address]
	return decimal, ok
}

func (d *DecimalsCache) Fetch(address common.Address, client *ethclient.Client, ctx context.Context) error {
	if _, ok := d.Get(address); ok {
		return nil
	}
	erc20Contract, err := erc20.NewErc20(address, client)
	if err != nil {
		return err
	}
	callOpts := &bind.CallOpts{
		Context: ctx,
	}
	dec, err := erc20Contract.Decimals(callOpts)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.decimals[address] = dec
	return nil
}

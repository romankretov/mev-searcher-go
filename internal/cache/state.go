package cache

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
	"time"
)

type PoolState struct {
	Name               string
	Reserve0, Reserve1 *big.Int
	BlockNumber        uint64
	UpdatedAt          time.Time
}

type StateCache struct {
	mu    sync.RWMutex
	pools map[common.Address]PoolState
}

func NewStateCache() *StateCache {
	return &StateCache{
		pools: make(map[common.Address]PoolState),
	}
}

func (c *StateCache) Set(address common.Address, state PoolState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pools[address] = state
}

func (c *StateCache) Get(address common.Address) (PoolState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	poolState, ok := c.pools[address]
	return poolState, ok
}

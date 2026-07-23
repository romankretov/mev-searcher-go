package target

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"searcher-bot/contracts/uniswapv2pair"
	"searcher-bot/internal/cache"
)

type PoolTarget struct {
	Name     string
	Address  common.Address
	Contract *uniswapv2pair.UniswapV2Pair
	Token0   common.Address
	Token1   common.Address
}

func LoadTargets(ctx context.Context, client *ethclient.Client, targets map[string]string, decimalsCache *cache.DecimalsCache) []PoolTarget {
	poolTargets := []PoolTarget{}
	for k, v := range targets {
		contract, err := uniswapv2pair.NewUniswapV2Pair(common.HexToAddress(v), client)
		if err != nil {
			continue
		}
		target := PoolTarget{
			Name:     k,
			Address:  common.HexToAddress(v),
			Contract: contract,
		}

		callOpts := &bind.CallOpts{
			Context: ctx,
		}
		token0, err := contract.Token0(callOpts)
		if err != nil {
			continue
		}
		token1, err := contract.Token1(callOpts)
		if err != nil {
			continue
		}
		target.Token0 = token0
		target.Token1 = token1
		poolTargets = append(poolTargets, target)
		if err := decimalsCache.Fetch(target.Token0, client, ctx); err != nil {
			continue
		}
		if err := decimalsCache.Fetch(target.Token1, client, ctx); err != nil {
			continue
		}

	}
	return poolTargets
}

package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"searcher-bot/contracts/uniswapv2pair"
	"searcher-bot/internal/cache"
	"searcher-bot/internal/target"
	"time"
)

type PoolTarget struct {
	Name     string
	Address  common.Address
	Contract *uniswapv2pair.UniswapV2Pair
	Token0   common.Address
	Token1   common.Address
}

func main() {
	const rpcUrl = "ws://localhost:8545"
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("error dialing anvil: %v", err)
	}
	defer client.Close()
	stateCache := cache.NewStateCache()
	decimalsCache := cache.NewDecimalsCache()

	headers := make(chan *types.Header)

	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatalf("error subscirbing to headers: %v", err)
	}
	defer sub.Unsubscribe()

	pools := map[string]string{
		"Uniswap V2 USDC/WETH": "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc",
		"Sushiswap USDC/WETH":  "0x397FF1542f962076d0BFE58eA045FfA2d347ACa0",
	}
	targetPool := target.LoadTargets(ctx, client, pools, decimalsCache)

	fmt.Println("Listenning to new blocks....")

	for {
		select {
		case header, ok := <-headers:
			if !ok {
				log.Println("Header channel is closed")
				return
			}
			if header == nil || header.Number == nil {
				log.Println("Invalid header received")
				continue
			}
			for _, pool := range targetPool {
				callCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				callOpts := &bind.CallOpts{
					Context:     callCtx,
					BlockNumber: header.Number,
				}
				reserves, err := pool.Contract.GetReserves(callOpts)
				cancel()
				if err != nil {
					log.Fatal("error fetching reserves")
				}
				stateCache.Set(
					pool.Address,
					cache.PoolState{
						Name:        pool.Name,
						Reserve0:    reserves.Reserve0,
						Reserve1:    reserves.Reserve1,
						BlockNumber: header.Number.Uint64(),
						UpdatedAt:   time.Now(),
					},
				)
			}
			for _, pool := range targetPool {
				state, ok := stateCache.Get(pool.Address)
				if !ok {
					log.Fatal("error fetching state from cache")
				}
				fmt.Printf("Name: %s, Reserve0: %s, Reserve1: %s, BlockNumber: %v, UpdatedAt: %s \n",
					state.Name, state.Reserve0.String(), state.Reserve1.String(),
					state.BlockNumber, state.UpdatedAt.Format(time.RFC3339))
			}
		case err := <-sub.Err():
			log.Fatalf("Subscribption error %v", err)

		}
	}

}

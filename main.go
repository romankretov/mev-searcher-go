package main

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"searcher-bot/internal/cache"
	"searcher-bot/internal/chain"
	"searcher-bot/internal/target"
)

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

	pools := map[string]string{
		"Uniswap V2 USDC/WETH": "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc",
		"Sushiswap USDC/WETH":  "0x397FF1542f962076d0BFE58eA045FfA2d347ACa0",
	}
	targetPool := target.LoadTargets(ctx, client, pools, decimalsCache)

	watcher := chain.NewWatcher(client, targetPool, stateCache)
	if err := watcher.Run(ctx); err != nil {
		log.Fatalf("failed to start a watcher")
	}

}

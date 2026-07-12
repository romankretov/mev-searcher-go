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
)

type PoolTarget struct {
	Name    string
	Address common.Address
}

func main() {
	const rpcUrl = "ws://localhost:8545"
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("error dialing anvil: %v", err)
	}
	defer client.Close()

	headers := make(chan *types.Header)

	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatalf("error subscirbing to headers: %v", err)
	}
	defer sub.Unsubscribe()

	pairAddress := common.HexToAddress("0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc")

	pair, err := uniswapv2pair.NewUniswapV2Pair(pairAddress, client)
	if err != nil {
		log.Fatalf("Error creating a pair %v", err)
	}

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
			callOpts := &bind.CallOpts{
				BlockNumber: header.Number,
				Context:     ctx,
			}
			reserves, err := pair.GetReserves(callOpts)
			if err != nil {
				log.Printf("Error fetching new blocks pair reserves: %v", err)
				continue
			}
			fmt.Printf(
				"block=%v, reserve0=%v, reserve1=%v, timestamp=%v \n",
				header.Number.String(), reserves.Reserve0.String(), reserves.Reserve1.String(), reserves.BlockTimestampLast,
			)
		case err := <-sub.Err():
			log.Fatalf("Subscribption error %v", err)

		}
	}

}

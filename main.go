package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"searcher-bot/contracts/uniswapv2pair"
	"sync"
	"time"
)

type PoolTarget struct {
	Name     string
	Address  common.Address
	Contract *uniswapv2pair.UniswapV2Pair
	Token0   common.Address
	Token1   common.Address
}

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

	pools := []PoolTarget{
		{Name: "Uniswap V2 USDC/WETH", Address: common.HexToAddress("0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc")},
		{Name: "Sushiswap USDC/WETH", Address: common.HexToAddress("0x397FF1542f962076d0BFE58eA045FfA2d347ACa0")},
	}

	for i := range pools {
		contract, err := uniswapv2pair.NewUniswapV2Pair(pools[i].Address, client)
		if err != nil {
			log.Fatal("error binding a contract")
		}

		pools[i].Contract = contract
		callOpts := &bind.CallOpts{
			Context: context.Background(),
		}
		token0, err := contract.Token0(callOpts)
		if err != nil {
			log.Fatal("error fetching token0")
		}
		pools[i].Token0 = token0

		token1, err := contract.Token1(callOpts)
		if err != nil {
			log.Fatal("error fetching token1")
		}
		pools[i].Token1 = token1
		fmt.Printf("Bound %s at %s (token0 = %s   token1 = %s) \n", pools[i].Name, pools[i].Address.Hex(), pools[i].Token0.Hex(), pools[i].Token1.Hex())
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
		case err := <-sub.Err():
			log.Fatalf("Subscribption error %v", err)

		}
	}

}

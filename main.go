package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"searcher-bot/contracts/uniswapv2pair"
	"time"
)

type PoolTarget struct {
	Name    string
	Address common.Address
}

func main() {
	const rpcUrl = "http://localhost:8545"
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatalf("Error connecting to the rpc")
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("Error fetching block number")
	}

	fmt.Printf("Connected to the RPC at %v \n", rpcUrl)
	fmt.Printf("Block number is %v  \n", blockNumber)

	pools := []PoolTarget{
		{
			Name:    "Uniswap V2 USDC/WETH",
			Address: common.HexToAddress("0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc"),
		},
		{
			Name:    "Sushiswap USDC/WETH",
			Address: common.HexToAddress("0x397FF1542f962076d0BFE58eA045FfA2d347ACa0"),
		},
	}
	for _, pool := range pools {
		printPool(ctx, client, pool)
	}

}

func printPool(ctx context.Context, client *ethclient.Client, pool PoolTarget) error {
	pair, err := uniswapv2pair.NewUniswapV2Pair(pool.Address, client)
	if err != nil {
		log.Fatal("Error creating new pair")
	}
	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	token0, err := pair.Token0(callOpts)
	if err != nil {
		log.Fatal("error loading token0")
	}
	token1, err := pair.Token1(callOpts)
	if err != nil {
		log.Fatal("error loading token1")
	}
	reservs, err := pair.GetReserves(callOpts)
	if err != nil {
		log.Fatal("error loading reservs")
	}
	fmt.Printf("Pool name: %v \n", pool.Name)
	fmt.Printf("Token 1: %v \n", token0.Hex())
	fmt.Printf("Token 2: %v \n", token1.Hex())
	fmt.Printf("Token 1 Reserves : %v \n", convertUnits(reservs.Reserve0, 6).FloatString(5))
	fmt.Printf("Token 2 Reserves: %v \n", convertUnits(reservs.Reserve1, 18).FloatString(5))
	fmt.Printf("Block timestamp: %v \n", reservs.BlockTimestampLast)
	ethRes := convertUnits(reservs.Reserve0, 6)
	wethRes := convertUnits(reservs.Reserve1, 18)
	price := new(big.Rat).Quo(ethRes, wethRes).FloatString(5)
	fmt.Println(price)

	return nil

}

func convertUnits(raw *big.Int, power int) *big.Rat {
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(power)), nil)
	return new(big.Rat).SetFrac(new(big.Int).Set(raw), divisor)

}

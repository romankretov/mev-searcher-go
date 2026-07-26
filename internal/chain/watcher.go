package chain

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"searcher-bot/contracts/uniswapv2pair"
	"searcher-bot/internal/cache"
	"searcher-bot/internal/target"
	"time"
)

type Watcher struct {
	client        *ethclient.Client
	pools         []target.PoolTarget
	stateCache    *cache.StateCache
	addressToPool map[common.Address]target.PoolTarget
}

func NewWatcher(client *ethclient.Client, pools []target.PoolTarget, cache *cache.StateCache) *Watcher {
	addressToPool := make(map[common.Address]target.PoolTarget, len(pools))
	for _, p := range pools {
		addressToPool[p.Address] = p
	}
	return &Watcher{
		client:        client,
		pools:         pools,
		stateCache:    cache,
		addressToPool: addressToPool,
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	startBlock, err := w.client.BlockNumber(ctx)
	if err != nil {
		log.Fatal("error fetching block bumber")
	}
	addresses := make([]common.Address, 0, len(w.pools))
	for _, p := range w.pools {
		addresses = append(addresses, p.Address)
	}

	w.reconcile(ctx, startBlock)
	syncSig := []byte("Sync(uint112,uint112)")
	syncTopic := crypto.Keccak256Hash(syncSig)

	query := ethereum.FilterQuery{
		Addresses: addresses,
		Topics:    [][]common.Hash{{syncTopic}},
	}

	const reconcileEvery = 5
	var blockCount uint64
	attempt := 0

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logs, subLog, headers, subHeader, err := w.subscribe(ctx, query)
		if err != nil {
			log.Printf("subscribe failed (attempt %v): %v \n", attempt+1, err)
			time.Sleep(nextRetry(attempt))
			attempt++
			continue
		}
		attempt = 0
		log.Println("listening to sync event")
		runErr := func() error {
			defer subLog.Unsubscribe()
			defer subHeader.Unsubscribe()

			for {
				select {
				case vLog, ok := <-logs:
					if !ok {
						return fmt.Errorf("log channel closed")
					}
					w.handleSyncLog(vLog)
				case header, ok := <-headers:
					if !ok {
						return fmt.Errorf("header channel closed")
					}
					blockCount++
					if blockCount%reconcileEvery == 0 {
						w.reconcile(ctx, header.Number.Uint64())
					}
				case err := <-subLog.Err():
					return fmt.Errorf("log subscription errror: %v", err)
				case err := <-subHeader.Err():
					return fmt.Errorf("header subscription error %v", err)
				case <-ctx.Done():
					return ctx.Err()

				}

			}
		}()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Printf("subscription dropped, reconnecting: %v \n", runErr)
		time.Sleep(nextRetry(attempt))
		attempt++

	}
}

func (w *Watcher) handleSyncLog(vLog types.Log) {
	pool, ok := w.addressToPool[vLog.Address]
	if !ok {
		log.Printf("Recieved sync event form untracked address: %v \n", vLog.Address.Hex())
		return
	}

	filterer, err := uniswapv2pair.NewUniswapV2PairFilterer(vLog.Address, nil)
	if err != nil {
		log.Println("Error creating a uniswap filterer")
		return
	}

	sync, err := filterer.ParseSync(vLog)
	if err != nil {
		log.Println("Errr parsing the event")
		return
	}
	w.stateCache.Set(
		pool.Address,
		cache.PoolState{
			Name:        pool.Name,
			Reserve0:    sync.Reserve0,
			Reserve1:    sync.Reserve1,
			BlockNumber: vLog.BlockNumber,
			UpdatedAt:   time.Now(),
		},
	)
	log.Printf("Sync: pool=%s, blocknumber=%v, reserve0=%v, reserve1=%v \n", pool.Name, vLog.BlockNumber, sync.Reserve0.String(), sync.Reserve1.String())

}

func (w *Watcher) reconcile(ctx context.Context, blockNumber uint64) {
	for _, p := range w.pools {
		callOpts := &bind.CallOpts{
			Context: ctx,
		}
		reserves, err := p.Contract.GetReserves(callOpts)
		if err != nil {
			log.Printf("Error fetching reserves on reconcile: %v \n", err)
		}
		w.stateCache.Set(
			p.Address,
			cache.PoolState{
				Name:        p.Name,
				Reserve0:    reserves.Reserve0,
				Reserve1:    reserves.Reserve1,
				BlockNumber: blockNumber,
				UpdatedAt:   time.Now(),
			},
		)
		log.Printf("reconciled: pool=%v, block=%v, reserve0=%v, reserve1=%v \n", p.Name, blockNumber, reserves.Reserve0.String(), reserves.Reserve1.String())
	}

}

func (w *Watcher) subscribe(ctx context.Context, query ethereum.FilterQuery) (
	logs chan types.Log, logSub ethereum.Subscription, headers chan *types.Header, headerSub ethereum.Subscription, err error,
) {
	logs = make(chan types.Log)
	logSub, err = w.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error subscribing to filter logs: %v", err)
	}

	headers = make(chan *types.Header)
	headerSub, err = w.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		logSub.Unsubscribe()
		return nil, nil, nil, nil, fmt.Errorf("error subscribing to new headers: %v", err)
	}

	return logs, logSub, headers, headerSub, nil
}

func nextRetry(attempt int) time.Duration {
	delay := time.Duration(1<<attempt) * time.Second
	if delay.Seconds() > 30 {
		return time.Duration(30)
	}
	return delay

}

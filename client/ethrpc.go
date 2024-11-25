package client

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/khanghh/ethcore/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Error = rpc.Error

type RemoteChainReader interface {
	// NetworkID returns the network identifier of the remote chain.
	NetworkID(ctx context.Context) (*big.Int, error)
	// BlockByHash retrieves a block from the remote chain by hash.
	BlockByHash(ctx context.Context, hash common.Hash, fullBlock bool) (*types.Block, error)
	// BlockByNumber retrieves a block from the remote chain by number.
	BlockByNumber(ctx context.Context, number *big.Int, fullBlock bool) (*types.Block, error)
	// BlockNumber retrieves the current block number of the remote chain.
	BlockNumber(ctx context.Context) (*big.Int, error)
	// GetTransactionByHash retrieves a transaction by its hash, also returns the block hash, block number, transaction index in block
	TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error)
	// TransactionReceipt retrieves the receipts of a transaction by its hash.
	TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error)
	// BlockReceipts retrieves the receipts of a block by its hash.
	BlockReceipts(ctx context.Context, numberOrHash interface{}) (types.Receipts, error)
	// CodeAt retrieves the contract code of the given account in the given block.
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
	// CallContract executes a contract call with the given parameters.
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	// Call executes an RPC call with the given method and arguments.
	Call(ctx context.Context, result interface{}, method string, args ...interface{}) error
	// BatchCall executes a batch of RPC calls.
	BatchCall(ctx context.Context, batch []rpc.BatchElem) error
	// Close closes the underlying RPC connections.
	Close()
}

func Dial(url string) (*ETHClient, error) {
	return DialContext(context.Background(), url)
}

func DialContext(ctx context.Context, url string) (*ETHClient, error) {
	client := &ETHClient{url: url}
	err := client.connect(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func SetupConnectionPool(urls []string) (*RpcConnectionPool, error) {
	clients := []*ETHClient{}
	lock := sync.Mutex{}
	sem := make(chan struct{}, 5)
	wg := sync.WaitGroup{}
	for _, url := range urls {
		sem <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer func() {
				wg.Done()
				<-sem
			}()
			ctx, cancel := context.WithTimeout(context.Background(), rpcDialTimeout)
			defer cancel()
			client, err := DialContext(ctx, url)
			if err != nil {
				log.Debug("Could not establish connection to RPC endpoint", "url", url, "err", err)
				return
			}
			log.Info("Connected to RPC endpoint", "url", url, "version", client.ClientVersion(), "latency", client.Latency())
			lock.Lock()
			clients = append(clients, client)
			lock.Unlock()
		}(url)
	}
	wg.Wait()
	if len(clients) > 0 {
		sort.Slice(clients, func(i, j int) bool {
			return clients[i].Latency() < clients[j].Latency()
		})
		return NewRpcConnectionPool(clients), nil
	}
	return nil, fmt.Errorf("no connection established")
}

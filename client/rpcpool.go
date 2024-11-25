package client

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/khanghh/ethcore/types"
)

const (
	rpcConnectionCooldown = 1 * time.Minute
	rpcDialTimeout        = 5 * time.Second
)

// RpcConnectionPool implements RemoteChainReader interface. It picks an ETHClient from pool
// to make RPC request, if client reate limit reached, it will put the client into cooldown
type RpcConnectionPool struct {
	clients []*ETHClient // List of RPC clients
	status  []int64      // Client status
	quitCh  chan struct{}
}

func (p *RpcConnectionPool) cooldown(idx int, client *ETHClient) {
	for {
		select {
		case <-p.quitCh:
			return
		case <-time.After(rpcConnectionCooldown):
			if err := client.connect(context.Background()); err != nil {
				log.Warn("Failed to reconnect to RPC", "url", client.url, "error", err)
			} else {
				atomic.StoreInt64(&p.status[idx], 0)
				return
			}
		}
	}
}

func (p *RpcConnectionPool) Close() {
	close(p.quitCh)
	for _, client := range p.clients {
		client.Close()
	}
}

func (p *RpcConnectionPool) Size() int {
	return len(p.clients)
}

func (p *RpcConnectionPool) NetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := p.Call(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

func (p *RpcConnectionPool) BlockByNumber(ctx context.Context, number *big.Int, fullBlock bool) (*types.Block, error) {
	return getBlock(ctx, p, "eth_getBlockByNumber", toBlockNumArg(number), fullBlock)
}

func (p *RpcConnectionPool) BlockNumber(ctx context.Context) (*big.Int, error) {
	var result string
	if err := p.Call(ctx, &result, "eth_blockNumber"); err != nil {
		return nil, err
	}
	return hexutil.DecodeBig(result)
}

func (p *RpcConnectionPool) BlockByHash(ctx context.Context, hash common.Hash, fullBlock bool) (*types.Block, error) {
	return getBlock(ctx, p, "eth_getBlockByHash", hash, fullBlock)
}

func (p *RpcConnectionPool) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	var tx *types.Transaction
	if err := p.Call(ctx, &tx, "eth_getTransactionByHash", hash); err != nil {
		return nil, false, err
	} else if tx == nil {
		return nil, false, ethereum.NotFound
	} else if _, r, _ := tx.RawSignatureValues(); r == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	return tx, tx.BlockNumber() == nil, nil
}

func (p *RpcConnectionPool) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	err := p.Call(ctx, &receipt, "eth_getTransactionReceipt", txHash)
	if err == nil && receipt == nil {
		return nil, ethereum.NotFound
	}
	return receipt, err
}

func (p *RpcConnectionPool) BlockReceipts(ctx context.Context, numberOrHash interface{}) (types.Receipts, error) {
	numberOrHashArg, err := parseNumberOrHash(numberOrHash)
	if err != nil {
		return nil, err
	}
	var result types.Receipts
	err = p.Call(ctx, &result, "eth_getBlockReceipts", numberOrHashArg)
	return result, err
}

func (p *RpcConnectionPool) CodeAt(ctx context.Context, account common.Address, number *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := p.Call(ctx, &result, "eth_getCode", account, toBlockNumArg(number))
	return result, err
}

func (p *RpcConnectionPool) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := p.Call(ctx, &hex, "eth_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (p *RpcConnectionPool) Call(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	var err error
	for idx, client := range p.clients {
		if !atomic.CompareAndSwapInt64(&p.status[idx], 0, 1) {
			continue
		}
		if err = client.Call(ctx, result, method, args...); err != nil {
			log.Warn("RPC request failed", "url", client.url, "method", method, "error", err)
			if err != ethereum.NotFound && err != rpc.ErrNoResult {
				go p.cooldown(idx, client)
			}
		} else {
			atomic.StoreInt64(&p.status[idx], 0)
			return nil
		}
	}
	return err
}

func (p *RpcConnectionPool) BatchCall(ctx context.Context, batch []rpc.BatchElem) error {
	var err error
	allBusy := true
	for idx, client := range p.clients {
		if !atomic.CompareAndSwapInt64(&p.status[idx], 0, 1) {
			continue
		}
		allBusy = false
		err = client.BatchCall(ctx, batch)
		if err == nil {
			err = getBatchErr(batch)
		}
		if err != nil {
			log.Warn("RPC batch request failed", "url", client.url, "count", len(batch), "error", err)
			go p.cooldown(idx, client)
		} else {
			atomic.StoreInt64(&p.status[idx], 0)
			return nil
		}
	}
	if allBusy {
		return fmt.Errorf("all clients are busy")
	}
	return err
}

func NewRpcConnectionPool(clients []*ETHClient) *RpcConnectionPool {
	return &RpcConnectionPool{
		clients: clients,
		status:  make([]int64, len(clients)),
		quitCh:  make(chan struct{}),
	}
}

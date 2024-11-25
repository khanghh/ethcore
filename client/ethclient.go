package client

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/khanghh/ethcore/types"

	"golang.org/x/exp/maps"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	rpcRequestBatchSize = 100
)

type ETHClient struct {
	url       string
	client    *rpc.Client
	networkId string
	version   string
	latency   time.Duration
}

func (ec *ETHClient) Url() string {
	return ec.url
}

func (ec *ETHClient) connect(ctx context.Context) error {
	client, err := rpc.DialContext(ctx, ec.url)
	if err != nil {
		return err
	}
	start := time.Now()
	if err := client.CallContext(ctx, &ec.networkId, "net_version"); err != nil {
		return err
	}
	ec.latency = time.Since(start)
	if err := client.CallContext(ctx, &ec.version, "web3_clientVersion"); err != nil {
		return err
	}
	ec.client = client
	return nil
}

func (ec *ETHClient) Latency() time.Duration {
	return ec.latency
}

func (ec *ETHClient) ClientVersion() string {
	return ec.version
}

func (ec *ETHClient) BlockByHash(ctx context.Context, hash common.Hash, fullBlock bool) (block *types.Block, reqErr error) {
	return getBlock(ctx, ec, "eth_getBlockByNumber", hash, fullBlock)
}

func (ec *ETHClient) BlockByNumber(ctx context.Context, number *big.Int, fullBlock bool) (*types.Block, error) {
	return getBlock(ctx, ec, "eth_getBlockByNumber", toBlockNumArg(number), fullBlock)
}

func (ec *ETHClient) BlockNumber(ctx context.Context) (*big.Int, error) {
	var result string
	if err := ec.client.CallContext(ctx, &result, "eth_blockNumber"); err != nil {
		return nil, err
	}
	return hexutil.DecodeBig(result)
}

func (ec *ETHClient) BlockReceipts(ctx context.Context, numberOrHash interface{}) (types.Receipts, error) {
	numberOrHashArg, err := parseNumberOrHash(numberOrHash)
	if err != nil {
		return nil, err
	}
	var result types.Receipts
	err = ec.Call(ctx, &result, "eth_getBlockReceipts", numberOrHashArg)
	return result, err
}

func (ec *ETHClient) CodeAt(ctx context.Context, account common.Address, number *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.Call(ctx, &result, "eth_getCode", account, toBlockNumArg(number))
	return result, err
}

func (ec *ETHClient) Close() {
	ec.client.Close()
}

func (ec *ETHClient) NetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := ec.Call(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

func (ec *ETHClient) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	var tx *types.Transaction
	if err := ec.Call(ctx, &tx, "eth_getTransactionByHash", hash); err != nil {
		return nil, false, err
	} else if tx == nil {
		return nil, false, ethereum.NotFound
	} else if _, r, _ := tx.RawSignatureValues(); r == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	return tx, tx.BlockNumber() == nil, nil
}

func (ec *ETHClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	err := ec.Call(ctx, &receipt, "eth_getTransactionReceipt", txHash)
	if err == nil && receipt == nil {
		return nil, ethereum.NotFound
	}
	return receipt, err
}

func (ec *ETHClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.Call(ctx, &hex, "eth_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (ec *ETHClient) Call(ctx context.Context, result interface{}, method string, params ...interface{}) error {
	err := ec.client.CallContext(ctx, result, method, params...)
	log.Debug("Request RPC call", "url", ec.url, "method", method, "params", params, "result", map[bool]string{true: "OK", false: fmt.Sprint(err)}[err == nil])
	return err
}

func (ec *ETHClient) BatchCall(ctx context.Context, batch []rpc.BatchElem) error {
	err := ec.client.BatchCallContext(ctx, batch)
	methodsMap := map[string]bool{}
	for _, elem := range batch {
		methodsMap[elem.Method] = true
	}
	log.Debug("Request RPC batch call", "url", ec.url, "count", len(batch), "methods", maps.Keys(methodsMap), "result", map[bool]string{true: "OK", false: fmt.Sprint(err)}[err == nil])
	return err
}

func NewClient(client *rpc.Client) *ETHClient {
	return &ETHClient{
		client: client,
	}
}

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/khanghh/ethcore/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type rpcCaller interface {
	Call(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCall(ctx context.Context, batch []rpc.BatchElem) error
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func parseNumberOrHash(numOrHash interface{}) (string, error) {
	switch v := numOrHash.(type) {
	case *big.Int:
		return toBlockNumArg(v), nil
	case common.Hash:
		return v.Hex(), nil
	default:
		return "", fmt.Errorf("invalid number or hash argument")
	}
}

func getBatchErr(batch []rpc.BatchElem) error {
	for _, elem := range batch {
		if elem.Error != nil {
			return elem.Error
		}
	}
	return nil
}

func getBlock(ctx context.Context, client rpcCaller, method string, numOrHash interface{}, fullBlock bool) (*types.Block, error) {
	var raw json.RawMessage
	if err := client.Call(ctx, &raw, method, numOrHash, fullBlock); err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.NotFound
	}
	var header *types.Header
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}
	if fullBlock {
		var resp struct {
			Uncles       []common.Hash      `json:"uncles"`
			Transactions types.Transactions `json:"transactions"`
			Withdrawals  types.Withdrawals  `json:"withdrawals"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, err
		}
		uncles, err := getBlockUncles(ctx, client, header.Hash, resp.Uncles)
		if err != nil {
			return nil, err
		}
		return types.NewBlockWithHeader(header).
			WithBody(resp.Transactions, uncles).
			WithWithdrawals(resp.Withdrawals), nil
	} else {
		var resp struct {
			Uncles       []common.Hash     `json:"uncles"`
			Transactions []common.Hash     `json:"transactions"`
			Withdrawals  types.Withdrawals `json:"withdrawals"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, err
		}
		return types.NewBlockWithHeader(header).
			WithCompactBody(resp.Transactions, resp.Uncles).
			WithWithdrawals(resp.Withdrawals), nil
	}
}

func getBlockUncles(ctx context.Context, client rpcCaller, blockHash common.Hash, hashes []common.Hash) ([]*types.Header, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	batch := make([]rpc.BatchElem, len(hashes))
	uncles := make([]*types.Header, len(hashes))
	for idx := range hashes {
		batch[idx] = rpc.BatchElem{
			Method: "eth_getUncleByBlockHashAndIndex",
			Args:   []interface{}{blockHash, hexutil.EncodeUint64(uint64(idx))},
			Result: &uncles[idx],
		}
	}
	if err := client.BatchCall(ctx, batch); err != nil {
		return nil, err
	}

	for idx, elem := range batch {
		if elem.Error != nil {
			return nil, elem.Error
		}
		if uncles[idx] == nil {
			return nil, fmt.Errorf("got null header for hash %s", hashes[idx])
		}
		if uncles[idx].Hash != hashes[idx] {
			return nil, fmt.Errorf("got wrong header for hash %s", hashes[idx])
		}
	}
	return uncles, nil
}

func batchGetTransactionReceipt(ctx context.Context, client rpcCaller, txHashes []common.Hash) (types.Receipts, error) {
	receipts := make(types.Receipts, len(txHashes))
	if len(txHashes) == 0 {
		return receipts, nil
	}
	batch := make([]rpc.BatchElem, len(txHashes))
	for i, txHash := range txHashes {
		batch[i] = rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txHash},
			Result: &receipts[i],
		}
	}
	if err := client.BatchCall(ctx, batch); err != nil {
		return nil, err
	}
	for idx, elem := range batch {
		if elem.Error != nil {
			return nil, elem.Error
		}
		if receipts[idx] == nil {
			return nil, fmt.Errorf("got null receipt for tx hash %s", txHashes[idx])
		}
	}
	return receipts, nil
}

func getBlockReceiptsByHashes(ctx context.Context, client rpcCaller, blockHash common.Hash, txHashes []common.Hash) (types.Receipts, error) {
	if len(txHashes) <= rpcRequestBatchSize {
		return batchGetTransactionReceipt(ctx, client, txHashes)
	} else {
		ret := make([]*types.Receipt, 0, len(txHashes))
		batchCount := (len(txHashes) + rpcRequestBatchSize - 1) / rpcRequestBatchSize
		for i := 0; i < batchCount; i++ {
			start := i * rpcRequestBatchSize
			end := start + rpcRequestBatchSize
			if end > len(txHashes) {
				end = len(txHashes)
			}
			receipts, err := batchGetTransactionReceipt(ctx, client, txHashes[start:end])
			if err != nil {
				return nil, err
			}
			ret = append(ret, receipts...)
		}
		for _, receipt := range ret {
			if receipt.BlockHash != blockHash {
				return nil, fmt.Errorf("receipt %s does not belong to block %s", receipt.TransactionHash, blockHash)
			}
		}
		return ret, nil
	}
}

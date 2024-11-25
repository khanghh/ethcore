package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:generate go run github.com/fjl/gencodec -type Receipt -field-override receiptMarshaling -out gen_receipt_json.go

const (
	// ReceiptStatusFailed is the status code of a transaction if execution failed.
	ReceiptStatusFailed = uint64(0)

	// ReceiptStatusSuccessful is the status code of a transaction if execution succeeded.
	ReceiptStatusSuccessful = uint64(1)
)

// Receipt represents the results of a transaction.
type Receipt struct {
	BlockHash         common.Hash     `json:"blockHash,omitempty"`
	BlockNumber       *big.Int        `json:"blockNumber,omitempty"`
	TransactionIndex  uint            `json:"transactionIndex"             gencodec:"required"`
	TransactionHash   common.Hash     `json:"transactionHash"              gencodec:"required"`
	Type              TransactionType `json:"type,omitempty"`
	ContractAddress   common.Address  `json:"contractAddress,omitempty"`
	GasUsed           uint64          `json:"gasUsed"                      gencodec:"required"`
	CumulativeGasUsed uint64          `json:"cumulativeGasUsed"            gencodec:"required"`
	EffectiveGasPrice *big.Int        `json:"effectiveGasPrice,omitempty"`
	From              common.Address  `json:"from"                         gencodec:"required"`
	To                *common.Address `json:"to,omitempty"`
	Logs              []*Log          `json:"logs"                         gencodec:"required"`
	LogsBloom         Bloom           `json:"logsBloom"                    gencodec:"required"`
	Status            uint64          `json:"status"                       gencodec:"required"`
}

type receiptMarshaling struct {
	BlockNumber       *hexutil.Big
	TransactionIndex  hexutil.Uint
	Type              hexutil.Uint64
	GasUsed           hexutil.Uint64
	CumulativeGasUsed hexutil.Uint64
	EffectiveGasPrice *hexutil.Big
	Status            hexutil.Uint64
}

// Receipts implements DerivableList for receipts.
type Receipts []*Receipt

// Len returns the number of receipts in this list.
func (rs Receipts) Len() int { return len(rs) }

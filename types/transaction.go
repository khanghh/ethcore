package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:generate go run github.com/fjl/gencodec -type txData -field-override txMarshaling -out gen_transaction_json.go
//go:generate go run github.com/fjl/gencodec -type AccessTuple -out gen_accesstuple_json.go

// Transaction types.
type TransactionType hexutil.Uint

const (
	LegacyTxType TransactionType = iota
	AccessListTxType
	DynamicFeeTxType
)

type AccessList []AccessTuple

// AccessTuple is the element type of an access list.
type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
}

// StorageKeys returns the total number of storage keys in the access list.
func (al AccessList) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

type txData struct {
	BlockHash   common.Hash     `json:"blockHash,omitempty"`                                // hash of the block where this transaction was in
	BlockNumber *big.Int        `json:"blockNumber,omitempty"`                              // number of the block where this transaction was in
	From        common.Address  `json:"from"                           gencodec:"required"` // sender addresss
	Gas         uint64          `json:"gas"                            gencodec:"required"` // gas used
	GasPrice    *big.Int        `json:"gasPrice"                       gencodec:"required"` // gas price in wei
	GasTipCap   *big.Int        `json:"maxPriorityFeePerGas,omitempty"`                     // a.k.a. maxPriorityFeePerGas
	GasFeeCap   *big.Int        `json:"maxFeePerGas,omitempty"`                             // a.k.a. maxFeePerGas
	Hash        common.Hash     `json:"hash"                           gencodec:"required"` // hash of the transaction
	Data        []byte          `json:"input"                          gencodec:"required"` // input data
	Nonce       uint64          `json:"nonce"                          gencodec:"required"` // nonce of sender account
	To          *common.Address `json:"to,omitempty"`                                       // recipient address
	TxIndex     uint            `json:"transactionIndex"               gencodec:"required"` // index of the transaction in the block
	Value       *big.Int        `json:"value"                          gencodec:"required"` // wei amount
	Type        TransactionType `json:"type"                           gencodec:"required"` // transaction type
	AccessList  AccessList      `json:"accessList,omitempty"`                               // access list
	ChainID     *big.Int        `json:"chainId,omitempty"`                                  // EIP 155 chain id
	V           *big.Int        `json:"v"                              gencodec:"required"` // signature values
	R           *big.Int        `json:"r"                              gencodec:"required"` // signature values
	S           *big.Int        `json:"s"                              gencodec:"required"` // signature values
}

type txMarshaling struct {
	BlockNumber *hexutil.Big
	Gas         hexutil.Uint64
	GasPrice    *hexutil.Big
	GasTipCap   *hexutil.Big
	GasFeeCap   *hexutil.Big
	Data        hexutil.Bytes
	Nonce       hexutil.Uint64
	TxIndex     hexutil.Uint64
	Type        hexutil.Uint64
	Value       *hexutil.Big
	ChainID     *hexutil.Big
	V, R, S     *hexutil.Big
}

type Transactions []*Transaction

func (s Transactions) Len() int { return len(s) }

type Transaction struct {
	data txData
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	return tx.data.MarshalJSON()
}

func (tx *Transaction) UnmarshalJSON(input []byte) error {
	return tx.data.UnmarshalJSON(input)
}

// BlockHash returns the hash of the block where the transaction was included in.
func (tx *Transaction) BlockHash() common.Hash { return tx.data.BlockHash }

// BlockNumber returns the number of the block where the transaction was included in.
func (tx *Transaction) BlockNumber() *big.Int { return tx.data.BlockNumber }

// From returns the sender address of the transaction.
func (tx *Transaction) From() common.Address { return tx.data.From }

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.data.Gas }

// GasPrice returns the gas price of the transaction.
func (tx *Transaction) GasPrice() *big.Int { return tx.data.GasPrice }

// GasTipCap returns max priority fee per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return tx.data.GasTipCap }

// GasFeeCap returns max fee per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return tx.data.GasFeeCap }

// Hash returns the hash of the transaction.
func (tx *Transaction) Hash() common.Hash { return tx.data.Hash }

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.data.Data }

// Nonce returns the nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.data.Nonce }

// To returns the recipient address of the transaction.
func (tx *Transaction) To() *common.Address { return tx.data.To }

// TxIndex returns the index of the transaction in the block.
func (tx *Transaction) TxIndex() uint { return tx.data.TxIndex }

// Value returns the amount of wei transferred in the transaction.
func (tx *Transaction) Value() *big.Int { return tx.data.Value }

// Type returns the type of the transaction.
func (tx *Transaction) Type() TransactionType { return tx.data.Type }

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList { return tx.data.AccessList }

// ChainID returns the chain ID of the transaction.
func (tx *Transaction) ChainID() *big.Int { return tx.data.ChainID }

func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

// Cost returns gas * gasPrice + value.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	total.Add(total, tx.Value())
	return total
}

// Message is a fully derived transaction and implements core.Message
type Message struct {
	from       common.Address
	to         *common.Address
	nonce      uint64
	amount     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	gasFeeCap  *big.Int
	gasTipCap  *big.Int
	data       []byte
	accessList AccessList
}

func (m Message) From() common.Address   { return m.from }
func (m Message) To() *common.Address    { return m.to }
func (m Message) GasPrice() *big.Int     { return m.gasPrice }
func (m Message) GasFeeCap() *big.Int    { return m.gasFeeCap }
func (m Message) GasTipCap() *big.Int    { return m.gasTipCap }
func (m Message) Value() *big.Int        { return m.amount }
func (m Message) Gas() uint64            { return m.gasLimit }
func (m Message) Nonce() uint64          { return m.nonce }
func (m Message) Data() []byte           { return m.data }
func (m Message) AccessList() AccessList { return m.accessList }

package types

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Body struct {
	Transactions Transactions
	Uncles       []*Header
}

func (b *Body) Transaction(hash common.Hash) *Transaction {
	for _, tx := range b.Transactions {
		if tx.Hash() == hash {
			return tx
		}
	}
	return nil
}

type compactBody struct {
	Transactions []common.Hash
	Uncles       []common.Hash
}

// CopyHeader creates a deep copy of a block header.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	if h.WithdrawalsHash != nil {
		cpy.WithdrawalsHash = new(common.Hash)
		*cpy.WithdrawalsHash = *h.WithdrawalsHash
	}
	if h.ExcessBlobGas != nil {
		cpy.ExcessBlobGas = new(uint64)
		*cpy.ExcessBlobGas = *h.ExcessBlobGas
	}
	if h.BlobGasUsed != nil {
		cpy.BlobGasUsed = new(uint64)
		*cpy.BlobGasUsed = *h.BlobGasUsed
	}
	if h.ParentBeaconRoot != nil {
		cpy.ParentBeaconRoot = new(common.Hash)
		*cpy.ParentBeaconRoot = *h.ParentBeaconRoot
	}
	return &cpy
}

type Blocks []*Block

type Block struct {
	header      *Header
	body        *Body
	compact     *compactBody
	withdrawals Withdrawals
}

// Accessors for body data. These do not return a copy because the content
// of the body slices does not affect the cached hash/size in block.

// Body returns the block body, compact block does not have body.
func (b *Block) Body() *Body { return b.body }

func (b *Block) Uncles() []common.Hash { return b.compact.Uncles }

func (b *Block) Transactions() []common.Hash { return b.compact.Transactions }

func (b *Block) Withdrawals() Withdrawals {
	return b.withdrawals
}

// Header returns the block header (as a copy).
func (b *Block) Header() *Header {
	return CopyHeader(b.header)
}

// Header value accessors. These do copy!

func (b *Block) Hash() common.Hash        { return b.header.Hash }
func (b *Block) Number() *big.Int         { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() uint64         { return b.header.GasLimit }
func (b *Block) GasUsed() uint64          { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int     { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() uint64             { return b.header.Time }
func (b *Block) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *Block) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *Block) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *Block) BeaconRoot() *common.Hash { return b.header.ParentBeaconRoot }

func (b *Block) ExcessBlobGas() *uint64 {
	var excessBlobGas *uint64
	if b.header.ExcessBlobGas != nil {
		excessBlobGas = new(uint64)
		*excessBlobGas = *b.header.ExcessBlobGas
	}
	return excessBlobGas
}

func (b *Block) BlobGasUsed() *uint64 {
	var blobGasUsed *uint64
	if b.header.BlobGasUsed != nil {
		blobGasUsed = new(uint64)
		*blobGasUsed = *b.header.BlobGasUsed
	}
	return blobGasUsed
}

func (b *Block) WithCompactBody(transactions []common.Hash, uncles []common.Hash) *Block {
	block := &Block{
		header: b.header,
		compact: &compactBody{
			Transactions: make([]common.Hash, len(transactions)),
			Uncles:       make([]common.Hash, len(uncles)),
		},
	}
	copy(block.compact.Transactions, transactions)
	copy(block.compact.Uncles, uncles)
	return block
}

func (b *Block) WithBody(transactions Transactions, uncles []*Header) *Block {
	block := &Block{
		header: b.header,
		body: &Body{
			Transactions: make(Transactions, len(transactions)),
			Uncles:       make([]*Header, len(uncles)),
		},
		compact: &compactBody{
			Transactions: make([]common.Hash, len(transactions)),
			Uncles:       make([]common.Hash, len(uncles)),
		},
	}

	for i, tx := range transactions {
		block.body.Transactions[i] = tx
		block.compact.Transactions[i] = tx.Hash()
	}
	for i, uncle := range uncles {
		block.compact.Uncles[i] = uncle.Hash
		block.body.Uncles[i] = CopyHeader(uncle)
	}

	return b
}

func (b *Block) WithWithdrawals(withdrawals Withdrawals) *Block {
	block := &Block{
		header:  b.header,
		body:    b.body,
		compact: b.compact,
	}
	if len(withdrawals) > 0 {
		block.withdrawals = make(Withdrawals, len(withdrawals))
		copy(block.withdrawals, withdrawals)
	}
	return block
}

func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: header}
}

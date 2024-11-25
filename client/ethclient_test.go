package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
)

func makeClient() *ETHClient {
	client, err := Dial("https://eth-pokt.nodies.app")
	if err != nil {
		panic(err)
	}
	return client
}

func TestGetBlockNumber(t *testing.T) {
	client := makeClient()
	num, err := client.BlockNumber(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(num)
}

// func TestGetBlockUncles(t *testing.T) {
// 	client := makeClient()

// }

func TestGetBlockByNumber(t *testing.T) {
	client := makeClient()
	block, err := client.BlockByNumber(context.Background(), big.NewInt(3340162), true)
	if err != nil {
		panic(err)
	}
	for _, uncle := range block.Body().Uncles {
		uncleJSON, _ := json.MarshalIndent(uncle, "", "  ")
		fmt.Println(string(uncleJSON))
	}
}

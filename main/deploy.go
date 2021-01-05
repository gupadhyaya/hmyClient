package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gupadhyaya/hmyClient/store"
	"github.com/harmony-one/harmony/accounts/abi/bind"
	"github.com/harmony-one/harmony/core/types"
)

type Client struct {
	c *rpc.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

// DialContext ...
func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{c}
}

// Close ...
func (ec *Client) Close() {
	ec.c.Close()
}

func toFilterArg(q ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{
		"address": q.Addresses,
		"topics":  q.Topics,
	}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}

func (ec *Client) HmySubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return ec.c.Subscribe(ctx, "hmy", channel, args...)
}

func (ec *Client) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	return ec.HmySubscribe(ctx, ch, "logs", arg)
}

func (ec *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	return ec.c.CallContext(ctx, nil, "hmy_sendRawTransaction", hexutil.Encode(data))
}

func (ec *Client) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "hmy_getCode", account, "pending")
	return result, err
}

func (ec *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	err := ec.c.CallContext(ctx, &hex, "hmy_estimateGas", toCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

func (ec *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := ec.c.CallContext(ctx, &hex, "hmy_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

func (ec *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := ec.c.CallContext(ctx, &result, "hmy_getTransactionCount", account, "pending")
	return uint64(result), err
}

func (ec *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var result []types.Log
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	err = ec.c.CallContext(ctx, &result, "hmy_getLogs", arg)
	return result, err
}

func (ec *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.c.CallContext(ctx, &hex, "hmy_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (ec *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := ec.c.CallContext(ctx, &result, "hmy_getCode", account, toBlockNumArg(blockNumber))
	return result, err
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

func main() {
	client, err := Dial("https://api.s0.b.hmny.io")
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA("<your private key>")
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	var gasPrice *big.Int
	gasPrice, err = client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(2))
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)     // in wei
	auth.GasLimit = uint64(300000) // in units
	auth.GasPrice = gasPrice

	fmt.Println(auth)

	input := "1.0"
	address, tx, instance, err := store.DeployStore(auth, client, input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(address.Hex())   // 0x147B8eb97fD247D06C4006D269c90C1908Fb5D54
	fmt.Println(tx.Hash().Hex()) // 0xdae8ba5444eefdc99f4d45cd0c4f24056cba6a02cefbf78066ef9f4188ff7dc0

	_ = instance
}

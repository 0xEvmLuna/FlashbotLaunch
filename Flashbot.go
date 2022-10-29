package flashbot

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// `eth_sendBundle` can be used to send your bundles to the Flashbots builder.
	MethodSendBundle = "eth_sendBundle"

	// eth_callBundle can be used to simulate a bundle against a specific block number,
	// including simulating a bundle at the top of the next block.
	MethodCallBundle = "eth_callBundle"

	// `eth_sendPrivateTransaction` used to send a single transaction to Flashbots.
	MethodSendPrivateTransaction = "eth_sendPrivateTransaction"

	// `eth_cancelPrivateTransaction` Method stops private
	// transactions from being submitted for future blocks.
	MethodCancelPrivateTransaction = "eth_cancelPrivateTransaction"

	MethodEstimateGasBundle = "eth_estimateGasBundle"
	MethodGetUserStats      = "flashbots_getUserStats"
	MethodGetBundleStats    = "flashbots_getBundleStats"
)

var (
	errorTransaction = errors.New("Transaction parameters are empty")
)

type FlashbotLaunch struct {
	Rpc        string
	PrivateKey *ecdsa.PrivateKey
}

type requestParams struct {
	JsonRPC string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// ############
//  sendBundle
// ############
type SendBundleParams struct {
	Transactions      []string `json:"txs"`
	BlockNumber       string   `json:"blockNumber"`
	MinTimestamp      int64    `json:"minTimestamp,omitempty"`
	MaxTimestamp      int64    `json:"maxTimestamp,omitempty"`
	RevertingTxHashes []string `json:"revertingTxHashes,omitempty"`
}

type SendBundleResponse struct {
	ID         uint          `json:"id"`
	Version    string        `json:"jsonrpc"`
	Result     *bundleResult `json:"result"`
	Raw        string
	StatusCode int
}

// ############
//  callBundle
// ############
type CallBundleParams struct {
	Transactions     []string `json:"txs"`
	BlockNumber      string   `json:"blockNumber"`
	StateBlockNumber string   `json:"stateBlockNumber"`
	Timestamp        int64    `json:"timestamp,omitempty"`
}

type CallBundleResponse struct {
	ID         uint         `json:"id"`
	Version    string       `json:"jsonrpc"`
	Result     *callResult  `json:"result"`
	Error      *errorResult `json:"error"`
	Raw        string
	StatusCode int
}

// ###########
//  userStats
// ###########
type UserStatsResponse struct {
	ID      uint       `json:"id"`
	Version string     `json:"jsonrpc"`
	Result  *userStats `json:"result"`
}

type userStats struct {
	IsHighPriority       bool   `json:"is_high_priority"`
	AllTimeMinerPayments string `json:"all_time_miner_payments"`
	AllTimeGasSimulated  string `json:"all_time_gas_simulated"`
	Last7dMinerPayments  string `json:"last_7d_miner_payments"`
	Last7dGasSimulated   string `json:"last_7d_gas_simulated"`
	Last1dMinerPayments  string `json:"last_1d_miner_payments"`
	Last1dGasSimulated   string `json:"last_1d_gas_simulated"`
}

type errorResult struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type bundleResult struct {
	BundleHash string `json:"bundleHash"`
}

// ###################
// transaction Result
// ###################
type txResult struct {
	CoinbaseDiff      string `json:"coinbaseDiff"`
	EthSentToCoinbase string `json:"ethSentToCoinbase"`
	FromAddress       string `json:"fromAddress"`
	GasFees           string `json:"gasFees"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           uint64 `json:"gasUsed"`
	ToAddress         string `json:"toAddress"`
	TxHash            string `json:"txHash"`
	Value             string `json:"value"`
	Error             string `json:"error,omitempty"`
}

type callResult struct {
	BundleGasPrice    string     `json:"bundleGasPrice"`
	BundleHash        string     `json:"bundleHash"`
	CoinbaseDiff      string     `json:"coinbaseDiff"`
	EthSentToCoinbase string     `json:"ethSentToCoinbase"`
	GasFees           string     `json:"gasFees"`
	Results           []txResult `json:"results"`
	StateBlockNumber  uint64     `json:"stateBlockNumber"`
	TotalGasUsed      uint64     `json:"totalGasUsed"`
}

func New(relayRPC string) *FlashbotLaunch {

	rpc, _ := RelayDefaultRPC(relayRPC)

	privateKey := os.Getenv("PrivateKey")
	if privateKey == "" {
		log.Fatal("The PrivateKey is nil!")
	}

	return &FlashbotLaunch{
		Rpc:        rpc,
		PrivateKey: HexToECDSA(privateKey),
	}
}

func (f *FlashbotLaunch) SendBundle(transactions []string, blockNumber uint64) (*SendBundleResponse, error) {
	if len(transactions) < 1 {
		return nil, errorTransaction
	}

	params := SendBundleParams{
		Transactions: transactions,
		BlockNumber:  WrapperBlockNumber(blockNumber),
	}

	sendBundle := &SendBundleResponse{}
	resp := f.requestRPC(MethodSendBundle, params)
	err := json.Unmarshal(resp, sendBundle)
	if err != nil {
		return nil, err
	}
	return sendBundle, nil
}

func (f *FlashbotLaunch) GetUserStats(blockNumber uint64) (*UserStatsResponse, error) {
	resp := f.requestRPC(MethodGetUserStats, blockNumber)
	print(string(resp))
	user := &UserStatsResponse{}
	err := json.Unmarshal(resp, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (f *FlashbotLaunch) requestRPC(Method string, params ...interface{}) []byte {
	request := requestParams{
		JsonRPC: "2.0",
		Id:      1,
		Method:  Method,
		Params:  append(params, params...),
	}
	fmt.Println("request", request)

	client := &http.Client{Timeout: 20 * time.Second}
	payload, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", f.Rpc, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal(err)
	}

	headerReady, _ := crypto.Sign(
		accounts.TextHash([]byte(hexutil.Encode(crypto.Keccak256(payload)))),
		f.PrivateKey,
	)

	signature := flashbotHeader(headerReady, f.PrivateKey)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Flashbots-Signature", signature)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	res, _ := ioutil.ReadAll(resp.Body)
	return res
}

func flashbotHeader(signature []byte, privateKey *ecdsa.PrivateKey) string {
	return crypto.PubkeyToAddress(privateKey.PublicKey).Hex() +
		":" + hexutil.Encode(signature)
}

func HexToECDSA(privateKey string) *ecdsa.PrivateKey {
	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

func WrapperBlockNumber(blockNumber uint64) string {
	return hexutil.EncodeUint64(blockNumber)
}

func RelayDefaultRPC(netType string) (string, error) {
	switch netType {
	case "mainnet":
		return "https://relay.flashbots.net", nil
	case "goerli":
		return "https://relay-goerli.flashbots.net", nil

	default:
		return "", fmt.Errorf("The netType is wrong!:", netType)
	}
}

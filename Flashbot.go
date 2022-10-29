
package Flashbot

import (
	"bytes"
	"encoding/json"
	"time"
	"net/http"
	"log"
	"crypto/ecdsa"
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/ethereum/go-ethereum/common/hexutil"
    "github.com/ethereum/go-ethereum/crypto"
)

type FlashbotLaunch struct {
	Rpc        string
	PrivateKey string
	Response   *http.Response
}

type requestParams struct {
	JsonRPC string 			`json: json_rpc`
	Id      int    			`json: id`
	Method  string 			`json: method`
	Params  []interface{}	
}

type SendBundleParams struct {
	Transactions      []string `json:"txs"`
	BlockNumber       string   `json:"blockNumber"`
	MinTimestamp      int64    `json:"minTimestamp,omitempty"`
	MaxTimestamp      int64    `json:"maxTimestamp,omitempty"`
	RevertingTxHashes []string `json:"revertingTxHashes,omitempty"`
}

type CallBundleParams struct {
	Transactions     []string `json:"txs"`
	BlockNumber      string   `json:"blockNumber"`
	StateBlockNumber string   `json:"stateBlockNumber"`
	Timestamp        int64    `json:"timestamp,omitempty"`
}

type SendBundleResponse struct {
	ID         uint          `json:"id"`
	Version    string        `json:"jsonrpc"`
	Result     *bundleResult `json:"result"`
	Raw        string
	StatusCode int
}

type CallBundleResponse struct {
	ID         uint         `json:"id"`
	Version    string       `json:"jsonrpc"`
	Result     *callResult  `json:"result"`
	Error      *errorResult `json:"error"`
	Raw        string
	StatusCode int
}

type UserStatsResponse struct {
	ID         uint       `json:"id"`
	Version    string     `json:"jsonrpc"`
	Result     *userStats `json:"result"`
	Raw        string
	StatusCode int
}

type errorResult struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type bundleResult struct {
	BundleHash string `json:"bundleHash"`
}

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

type userStats struct {
	IsHighPriority       bool   `json:"is_high_priority"`
	AllTimeMinerPayments string `json:"all_time_miner_payments"`
	AllTimeGasSimulated  string `json:"all_time_gas_simulated"`
	Last7dMinerPayments  string `json:"last_7d_miner_payments"`
	Last7dGasSimulated   string `json:"last_7d_gas_simulated"`
	Last1dMinerPayments  string `json:"last_1d_miner_payments"`
	Last1dGasSimulated   string `json:"last_1d_gas_simulated"`
}

func (f *FlashbotLaunch) SendBundle() {
	params := SendBundleParams{
		
	}
	f.requestRPC("flashbots_sendBundle", )
}

func New(relayRPC string) *FlashbotLaunch {
	if relayRPC == "" {
		relayRPC, _ := RelayDefaultRPC("mainnet")
	}
	http.DefaultClient
	privateKey := os.Getenv("PrivateKey")
	if privateKey == "" {
		log.Fatal("The PrivateKey is nil!")
	}
	mevHTTPClient := &http.Client{
		Timeout: time.Second * 5,
	}
	mevHTTPClient.Do()
	return &FlashbotLaunch{
		Rpc: 		relayRPC,
		PrivateKey:	privateKey
	}
}

func (f *FlashbotLaunch) requestRPC(method string, params ...interface{}) {
	request := requestParams{
		JsonRPC: "2.0",
		Id:		 1,
		Method:	 method,
		Params:  append([]interface{}, params...)
	}

	
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
	
	f.Response, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
}

func flashbotHeader(signature []byte, privateKey *ecdsa.PrivateKey) string {
    return crypto.PubkeyToAddress(privateKey.PublicKey).Hex() +
        ":" + hexutil.Encode(signature)
}

func RelayDefaultRPC(string netType) (string, error) {
	switch netType {
	case "mainnet":
		return "https://relay.flashbots.net", nil
	case "goerli":
		return "https://relay-goerli.flashbots.net", nil

	default:
		return nil, error.Errorf("The netType is wrong!:", netType)
	}
}

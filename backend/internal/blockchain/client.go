package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrClientDisabled = errors.New("blockchain: client is disabled or unconfigured")
	ErrNodeOffline    = errors.New("blockchain: node is offline or unreachable")
)

// Client defines the operations exposed by the blockchain ledger layer.
type Client interface {
	IsEnabled() bool
	RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) (string, error)
	RecordConsent(ctx context.Context, requestID, patientID, hospitalID uuid.UUID, status uint8, metadataHash string) (string, error)
	RecordDataAccess(ctx context.Context, requestID, actorID uuid.UUID, dataHash string) (string, error)
	VerifyProfileHash(ctx context.Context, patientID uuid.UUID, currentHash string) (bool, string, error)
}

type Config struct {
	RPCURL          string
	ContractAddress string
	FromAddress     string
}

type rpcClient struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	fromAddr   string
}

// NewClient returns an EVM RPC blockchain client. If RPCURL or ContractAddress
// is empty, it returns a disabled no-op/fallback client.
func NewClient(cfg Config) Client {
	if strings.TrimSpace(cfg.RPCURL) == "" || strings.TrimSpace(cfg.ContractAddress) == "" {
		return &noopClient{}
	}

	return &rpcClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 4 * time.Second, // Fast timeout for responsive fallback
		},
		fromAddr: cfg.FromAddress,
	}
}

func (c *rpcClient) IsEnabled() bool {
	return true
}

func (c *rpcClient) RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) (string, error) {
	patientB32 := UUIDToBytes32(patientID)
	hashB32, err := HexToBytes32(profileHash)
	if err != nil {
		return "", fmt.Errorf("parse profile hash: %w", err)
	}

	data := EncodeRecordProfileHash(patientB32, hashB32)
	return c.sendTransaction(ctx, data)
}

func (c *rpcClient) RecordConsent(
	ctx context.Context,
	requestID, patientID, hospitalID uuid.UUID,
	status uint8,
	metadataHash string,
) (string, error) {
	reqB32 := UUIDToBytes32(requestID)
	patB32 := UUIDToBytes32(patientID)
	hospB32 := UUIDToBytes32(hospitalID)
	metaB32, _ := HexToBytes32(metadataHash)

	data := EncodeRecordConsentDecision(reqB32, patB32, hospB32, status, metaB32)
	return c.sendTransaction(ctx, data)
}

func (c *rpcClient) RecordDataAccess(
	ctx context.Context,
	requestID, actorID uuid.UUID,
	dataHash string,
) (string, error) {
	reqB32 := UUIDToBytes32(requestID)
	actorB32 := UUIDToBytes32(actorID)
	dHashB32, _ := HexToBytes32(dataHash)

	data := EncodeRecordDataAccess(reqB32, actorB32, dHashB32)
	return c.sendTransaction(ctx, data)
}

func (c *rpcClient) VerifyProfileHash(ctx context.Context, patientID uuid.UUID, currentHash string) (bool, string, error) {
	patientB32 := UUIDToBytes32(patientID)
	hashB32, err := HexToBytes32(currentHash)
	if err != nil {
		return false, "", fmt.Errorf("parse current hash: %w", err)
	}

	data := EncodeVerifyProfileHash(patientB32, hashB32)

	res, err := c.ethCall(ctx, data)
	if err != nil {
		return false, "", err
	}

	return DecodeVerifyProfileHashResult(res)
}

// --- JSON-RPC Helpers ---

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int    `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *rpcClient) sendTransaction(ctx context.Context, data string) (string, error) {
	from, err := c.resolveFromAddress(ctx)
	if err != nil {
		return "", err
	}

	txObj := map[string]string{
		"from": from,
		"to":   c.cfg.ContractAddress,
		"data": data,
	}

	var txHash string
	if err := c.callRPC(ctx, "eth_sendTransaction", []any{txObj}, &txHash); err != nil {
		return "", err
	}

	return txHash, nil
}

func (c *rpcClient) ethCall(ctx context.Context, data string) (string, error) {
	callObj := map[string]string{
		"to":   c.cfg.ContractAddress,
		"data": data,
	}

	var res string
	if err := c.callRPC(ctx, "eth_call", []any{callObj, "latest"}, &res); err != nil {
		return "", err
	}
	return res, nil
}

func (c *rpcClient) resolveFromAddress(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.fromAddr != "" {
		return c.fromAddr, nil
	}

	var accounts []string
	if err := c.callRPC(ctx, "eth_accounts", []any{}, &accounts); err != nil {
		return "", fmt.Errorf("query eth_accounts: %w", err)
	}
	if len(accounts) == 0 {
		return "", errors.New("no unlocked Ethereum accounts found on node")
	}

	c.fromAddr = accounts[0]
	return c.fromAddr, nil
}

func (c *rpcClient) callRPC(ctx context.Context, method string, params []any, out any) error {
	reqBody, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.RPCURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNodeOffline, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read rpc response: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return fmt.Errorf("unmarshal rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if out != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("unmarshal rpc result into target: %w", err)
		}
	}

	return nil
}

// --- No-Op Fallback Client (when blockchain RPC is not enabled or offline) ---

type noopClient struct{}

func (n *noopClient) IsEnabled() bool {
	return false
}

func (n *noopClient) RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) (string, error) {
	log.Printf("[blockchain] notice: blockchain disabled; hash recorded in local cryptographic ledger only")
	return "", nil
}

func (n *noopClient) RecordConsent(ctx context.Context, requestID, patientID, hospitalID uuid.UUID, status uint8, metadataHash string) (string, error) {
	log.Printf("[blockchain] notice: blockchain disabled; consent recorded in local cryptographic ledger only")
	return "", nil
}

func (n *noopClient) RecordDataAccess(ctx context.Context, requestID, actorID uuid.UUID, dataHash string) (string, error) {
	log.Printf("[blockchain] notice: blockchain disabled; data access recorded in local cryptographic ledger only")
	return "", nil
}

func (n *noopClient) VerifyProfileHash(ctx context.Context, patientID uuid.UUID, currentHash string) (bool, string, error) {
	return false, "", ErrClientDisabled
}

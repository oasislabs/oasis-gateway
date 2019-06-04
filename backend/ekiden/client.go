package ekiden

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/ekiden"
	"github.com/oasislabs/developer-gateway/errors"
	tx "github.com/oasislabs/developer-gateway/tx/core"
)

type NodeProps struct {
	URL string
}

type ClientProps struct {
	PrivateKeys     []*ecdsa.PrivateKey
	RuntimeID       []byte
	RuntimeProps    NodeProps
	KeyManagerProps NodeProps
}

type Client struct {
	runtime    *ekiden.Runtime
	keyManager *ekiden.Enclave
	runtimeID  []byte
	handler    tx.TransactionHandler
}

func DialContext(ctx context.Context, props ClientProps) (*Client, errors.Err) {
	runtime, err := ekiden.DialRuntimeContext(ctx, props.RuntimeProps.URL)
	if err != nil {
		return nil, errors.New(errors.ErrEkidenDial, err)
	}

	keyManager, err := ekiden.DialEnclaveContext(ctx, &ekiden.EnclaveProps{
		URL:      props.KeyManagerProps.URL,
		Endpoint: "key-manager",
	})
	if err != nil {
		return nil, errors.New(errors.ErrEkidenDial, err)
	}

	return &Client{
		runtime:    runtime,
		keyManager: keyManager,
		runtimeID:  props.RuntimeID,
		handler:    props.Handler,
	}, nil
}

func (c *Client) GetPublicKeyService(
	ctx context.Context,
	req core.GetPublicKeyServiceRequest,
) (*core.GetPublicKeyServiceResponse, errors.Err) {
	decoded, err := hex.DecodeString(req.Address)
	if err != nil {
		return nil, errors.New(errors.ErrInvalidAddress, err)
	}

	if len(decoded) != 20 {
		return nil, errors.New(errors.ErrInvalidAddress, nil)
	}

	var address ekiden.Address
	copy(address[:], decoded)

	_, err = c.keyManager.GetPublicKey(ctx, &ekiden.GetPublicKeyRequest{
		Address: address,
	})
	if err != nil {
		return nil, errors.New(errors.ErrEkidenGetPublicKey, err)
	}

	return &core.GetPublicKeyServiceResponse{}, nil
}

func (c *Client) ExecuteService(
	ctx context.Context,
	id uint64,
	req core.ExecuteServiceRequest,
) (*core.ExecuteServiceResponse, errors.Err) {
	if err := c.submitTx(ctx, req.Address, req.Data); err != nil {
		return nil, err
	}

	return &core.ExecuteServiceResponse{
		ID:      id,
		Address: req.Address,
		Output:  "",
	}, nil
}

func (c *Client) DeployService(
	ctx context.Context,
	id uint64,
	req core.DeployServiceRequest,
) (*core.DeployServiceResponse, errors.Err) {
	if err := c.submitTx(ctx, "", req.Data); err != nil {
		return nil, err
	}

	// TODO(stan): get address
	return &core.DeployServiceResponse{
		ID:      id,
		Address: "",
	}, nil
}

func (c *Client) SubscribeRequest(
	ctx context.Context,
	id uint64,
	req core.SubscribeRequest,
	ch chan<- interface{},
) errors.Err {
	return errors.New(errors.ErrAPINotImplemented, nil)
}

func (c *Client) generateTx(ctx context.Context, transaction *types.Transaction) ([]byte, errors.Err) {
	tx, err := c.handler.Sign(ctx, tx.SignRequest{Transaction: transaction})
	if err != nil {
		return nil, errors.New(errors.ErrEkidenSignTx, err)
	}

	buffer := bytes.NewBuffer(make([]byte, 0, 16))
	if err := tx.EncodeRLP(buffer); err != nil {
		return nil, errors.New(errors.ErrEkidenEncodeRLPTx, err)
	}

	return buffer.Bytes(), nil
}

func (c *Client) createTx(address string, data string) *types.Transaction {
	gas := uint64(1000000)
	gasPrice := int64(1000000000)

	if len(address) == 0 {
		return types.NewContractCreation(0,
			big.NewInt(0), gas, big.NewInt(gasPrice), []byte(data))
	} else {
		return types.NewTransaction(0, common.HexToAddress(address),
			big.NewInt(0), gas, big.NewInt(gasPrice), []byte(data))
	}
}

func (c *Client) submitTx(ctx context.Context, address, data string) errors.Err {
	tx := c.createTx(address, data)
	p, err := c.generateTx(ctx, tx)
	if err != nil {
		return err
	}

	_, derr := c.runtime.EthereumTransaction(ctx, &ekiden.EthereumTransactionRequest{
		RuntimeID: c.runtimeID,
		Data:      p,
	})
	if derr != nil {
		return errors.New(errors.ErrEkidenSubmitTx, derr)
	}

	return nil
}

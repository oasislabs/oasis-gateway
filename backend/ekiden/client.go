package ekiden

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"math/big"

	cbor "bitbucket.org/bodhisnarkva/cbor/go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/ekiden/go/grpc/txnscheduler"
	"google.golang.org/grpc"
)

type submitTxRequest struct {
	Method string `cbor:"method"`
	Args   []byte `cbor:"args"`
}

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
}

type ClientProps struct {
	NodeProps
	Wallet
}

type Client struct {
	discovery *Discovery
	signer    types.Signer
	wallet    Wallet
	props     NodeProps
}

func NewClient(discovery *Discovery, props ClientProps) *Client {
	if discovery == nil {
		panic("discovery must be set")
	}

	return &Client{
		discovery: discovery,
		signer:    types.FrontierSigner{},
		wallet:    props.Wallet,
		props:     props.NodeProps}
}

func (c *Client) GetPublicKeyService(
	ctx context.Context,
	req core.GetPublicKeyServiceRequest,
) (*core.GetPublicKeyServiceResponse, errors.Err) {
	return nil, errors.New(errors.ErrAPINotImplemented, nil)
}

func (c *Client) ExecuteService(
	ctx context.Context,
	id uint64,
	req core.ExecuteServiceRequest,
) (*core.ExecuteServiceResponse, errors.Err) {
	conn, err := c.discovery.Conn(ctx, Compute)
	if err != nil {
		return nil, err
	}

	if err := c.submitTx(ctx, conn, req.Address, req.Data); err != nil {
		return nil, err
	}

	return nil, errors.New(errors.ErrAPINotImplemented, nil)
}

func (c *Client) generateTx(tx *types.Transaction) ([]byte, errors.Err) {
	tx, err := types.SignTx(tx, c.signer, c.wallet.PrivateKey)
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

func (c *Client) submitTx(ctx context.Context, conn *grpc.ClientConn, address, data string) errors.Err {
	tx := c.createTx(address, data)
	p, err := c.generateTx(tx)
	if err != nil {
		return err
	}

	msg := submitTxRequest{
		Method: "ethereum_transaction",
		Args:   p,
	}

	payload, derr := cbor.Dumps(msg)
	if err != nil {
		return errors.New(errors.ErrEkidenEncodeTx, derr)
	}

	sched := txnscheduler.NewTransactionSchedulerClient(conn)
	_, derr = sched.SubmitTx(ctx, &txnscheduler.SubmitTxRequest{
		RuntimeId: c.discovery.RuntimeID(),
		Data:      payload,
	})

	if err != nil {
		return errors.New(errors.ErrEkidenSubmitTx, derr)
	}

	// h, derr := hexutil.Decode(tx.Hash().Hex())
	// if derr != nil {
	// 	fmt.Println("FAILEWD TO DECODE HEX")
	// 	return errors.New(errors.ErrEkidenSubmitTx, derr)
	// }

	// for {
	// 	res, derr := sched.IsTransactionQueued(ctx, &txnscheduler.IsTransactionQueuedRequest{
	// 		RuntimeId: c.discovery.RuntimeID(),
	// 		Hash:      h,
	// 	})
	// 	if derr != nil {
	// 		return errors.New(errors.ErrEkidenSubmitTx, derr)
	// 	}

	// 	fmt.Println("RESPONSE: ", res.IsQueued)
	// }

	return nil
}

func (c *Client) DeployService(
	ctx context.Context,
	id uint64,
	req core.DeployServiceRequest,
) (*core.DeployServiceResponse, errors.Err) {
	return nil, errors.New(errors.ErrAPINotImplemented, nil)
}

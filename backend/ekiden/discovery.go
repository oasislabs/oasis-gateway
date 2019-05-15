package ekiden

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	stderr "errors"
	"fmt"
	"net"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/ekiden/go/grpc/common"
	"github.com/oasislabs/ekiden/go/grpc/registry"
	"github.com/oasislabs/ekiden/go/grpc/scheduler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type CommitteeKind string

const (
	Storage CommitteeKind = "STORAGE"
	Compute CommitteeKind = "COMPUTE"
)

type NodeProps struct {
	URL       string
	TLSConfig *tls.Config
}

type DiscoveryProps struct {
	RuntimeID []byte
	Registry  NodeProps
	Scheduler NodeProps
}

type Discovery struct {
	runtimeID []byte

	registry  registry.EntityRegistryClient
	scheduler scheduler.SchedulerClient
}

func getGrpcTransport(config *tls.Config) grpc.DialOption {
	if config == nil {
		return grpc.WithInsecure()

	} else {
		creds := credentials.NewTLS(config)
		return grpc.WithTransportCredentials(creds)
	}
}

func dialGrpcConn(ctx context.Context, props NodeProps) (*grpc.ClientConn, errors.Err) {
	transport := getGrpcTransport(props.TLSConfig)
	conn, err := grpc.DialContext(ctx, props.URL, transport)
	if err != nil {
		return nil, errors.New(errors.ErrEkidenDial, err)
	}

	return conn, nil
}

func parseCertificate(p []byte) (*x509.Certificate, errors.Err) {
	cert, err := x509.ParseCertificate(p)
	if err != nil {
		return nil, errors.New(errors.ErrEkidenParseCertificate, err)
	}

	return cert, nil
}

func constructAddress(address *common.Address) (string, errors.Err) {
	if address.Port > 65535 {
		return "", errors.New(errors.ErrEkidenPortInvalid, nil)
	}

	switch address.Transport {
	case common.Address_TCPv4:
		if len(address.Address) != 4 {
			return "", errors.New(errors.ErrEkidenAddressInvalid, nil)
		}

		return fmt.Sprintf("%s:%d", net.IP(address.Address), address.Port), nil
	case common.Address_TCPv6:
		if len(address.Address) != 8 {
			return "", errors.New(errors.ErrEkidenAddressInvalid, nil)
		}

		return fmt.Sprintf("%s:%d", net.IP(address.Address), address.Port), nil
	default:
		return "", errors.New(errors.ErrEkidenAddressTransportUnsupported, nil)
	}
}

func DialContext(ctx context.Context, props DiscoveryProps) (*Discovery, errors.Err) {
	registryConn, err := dialGrpcConn(ctx, props.Registry)
	if err != nil {
		return nil, err
	}

	schedulerConn, err := dialGrpcConn(ctx, props.Scheduler)
	if err != nil {
		return nil, err
	}

	return &Discovery{
		// runtimeID: props.RuntimeID,
		runtimeID: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		registry:  registry.NewEntityRegistryClient(registryConn),
		scheduler: scheduler.NewSchedulerClient(schedulerConn),
	}, nil
}

func (d *Discovery) Conn(ctx context.Context, kind CommitteeKind) (*grpc.ClientConn, errors.Err) {
	node, err := d.CommitteeLeader(ctx, kind)
	if err != nil {
		return nil, err
	}

	cert, err := parseCertificate(node.Certificate.Der)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(cert)

	// TODO(stan): handle node addresses correctly.
	if len(node.Addresses) == 0 {
		return nil, errors.New(errors.ErrEkidenNodeNoAddress, err)
	}

	address, err := constructAddress(node.Addresses[0])
	if err != nil {
		return nil, err
	}

	// TODO(stan): pool addresses, track its lifetime, and close when expired
	conn, err := dialGrpcConn(ctx, NodeProps{
		URL: address,
		TLSConfig: &tls.Config{
			RootCAs:    certPool,
			ServerName: "ekiden-node",
		},
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (d *Discovery) RuntimeID() []byte {
	return d.runtimeID
}

func (d *Discovery) CommitteeLeader(ctx context.Context, kind CommitteeKind) (*common.Node, errors.Err) {
	committee, err := d.Committee(ctx, kind)
	if err != nil {
		return nil, err
	}

	for _, node := range committee.Members {
		if node.Role == scheduler.CommitteeNode_LEADER {
			res, err := d.registry.GetNode(ctx, &registry.NodeRequest{Id: node.PublicKey})
			if err != nil {
				return nil, errors.New(errors.ErrEkidenGetNode, err)
			}

			return res.Node, nil
		}
	}

	return nil, errors.New(errors.ErrEkidenCommitteeLeaderNotFound, stderr.New("could not find commitee leader"))
}

func (d *Discovery) Committee(ctx context.Context, kind CommitteeKind) (*scheduler.Committee, errors.Err) {
	value, ok := scheduler.Committee_Kind_value[string(kind)]
	if !ok {
		return nil, errors.New(errors.ErrEkidenCommitteeKindUndefined, stderr.New("committe kind does not exist"))
	}

	req := scheduler.CommitteeRequest{
		RuntimeId: d.runtimeID,
	}

	res, err := d.scheduler.GetCommittees(ctx, &req)
	if err != nil {
		return nil, errors.New(errors.ErrEkidenGetCommittee, err)
	}

	for _, committee := range res.Committee {
		if committee.Kind == scheduler.Committee_Kind(value) {
			return committee, nil
		}
	}

	return nil, errors.New(errors.ErrEkidenCommitteeNotFound, stderr.New("could not find committee"))
}

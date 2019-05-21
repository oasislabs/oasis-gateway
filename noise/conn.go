package noise

import (
	"bytes"
	"context"
	"errors"
	"io"
)

// Requester represents a channel between the local endpoint
// and the remote endpoint that a Conn uses to abstract
// the underlying transport
type Requester interface {
	// Request abstracts a request into a reader which contains the
	// request, and a writer, where the response will be written
	Request(context.Context, io.Writer, io.Reader) error
}

// ClientFunc allows functions to implement a client
type ClientFunc func(context.Context, io.Writer, io.Reader) error

// Client implementation for ClientFunc
func (fn ClientFunc) Request(ctx context.Context, w io.Writer, r io.Reader) error {
	return fn(ctx, w, r)
}

// Conn represents a noise connection to a remote endpoint. It abstracts
// handling of the session state. A connection is not concurrency safe,
// if that's a needed property looked into using a FixedConnPool.
// A Conn does not represent a real network connection, it's an abstraction
// which uses a client to create the illusion of a Conn, it's the Client
// implementation what defines the underlying model. A Conn allows mutliplexing
// of multiple sessions over the same networking connection.
type Conn struct {
	client  Requester
	session *Session

	in  *bytes.Buffer
	out *bytes.Buffer
}

// DialConnContext creates a new connection and completes the handshake with the
// remote endpoint. If the handshake fails the Dial will also be
// considered failed
func DialConnContext(ctx context.Context, client Requester, props *SessionProps) (*Conn, error) {
	if !props.Initiator {
		return nil, errors.New("when dialing the connection has to initiate the handshake")
	}

	session, err := NewSession(props)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		client:  client,
		session: session,
		in:      bytes.NewBuffer(make([]byte, 0, 512)),
		out:     bytes.NewBuffer(make([]byte, 0, 512)),
	}
	if err := conn.doHandshake(ctx); err != nil {
		return nil, err
	}

	return conn, nil
}

// Request issues a request in the reader and writes the
// received response back to the writer
func (c *Conn) Request(ctx context.Context, req RequestPayload) (ResponsePayload, error) {
	return c.request(ctx, req)
}

// sendFrame sends a frame to the remote endpoint. This method should
// be used during the handshake and res, req should have content only
// if payloads are expected from the remote and local endpoint
func (c *Conn) sendFrame(ctx context.Context, res io.Writer, req io.Reader) error {
	// encrypt the request contents with the ciphers in the session
	if _, err := c.session.Write(c.in, req); err != nil {
		return err
	}

	// serialize the contents of the buffer along with the session ID
	// into a frame that can finally be sent on the network
	if err := SerializeIntoFrame(c.out, c.in, c.session.ID()); err != nil {
		return err
	}

	// send request whose contents are in c.out. At this point
	// c.in should be empty
	if err := c.client.Request(ctx, c.in, c.out); err != nil {
		return err
	}

	// decrypt session contents
	_, err := c.session.Read(res, c.in)
	return err
}

// request sends a full request with payload to the remote endpoint. A
// request can be send after the handshake has been completed. During
// the handshake sendFrame should be used to send frames without
// extra content
func (c *Conn) request(ctx context.Context, req RequestPayload) (ResponsePayload, error) {
	// wrap the request bytes into a RequestMessage
	if err := SerializeRequestMessage(c.in, &OutgoingRequestMessage{
		Request: req,
	}); err != nil {
		return ResponsePayload{}, err
	}

	// encrypt the request contents with the ciphers in the session
	if _, err := c.session.Write(c.out, c.in); err != nil {
		return ResponsePayload{}, err
	}

	// serialize the contents of the buffer along with the session ID
	// into a frame that can finally be sent on the network
	if err := SerializeIntoFrame(c.in, c.out, c.session.ID()); err != nil {
		return ResponsePayload{}, err
	}

	// send request whose contents are in c.out. At this point
	// c.in should be empty
	if err := c.client.Request(ctx, c.out, c.in); err != nil {
		return ResponsePayload{}, err
	}

	// decrypt session contents
	_, err := c.session.Read(c.in, c.out)
	if err != nil {
		return ResponsePayload{}, err
	}

	var res ResponseMessage
	// parse the response to make sure that it's a ResponseMessage
	if err := DeserializeResponseMessage(c.in, &res); err != nil {
		return ResponsePayload{}, err
	}

	return res.Response, nil
}

// doHandshake performs the initial handshake with the remote endpoint
func (c *Conn) doHandshake(ctx context.Context) error {
	in := bytes.NewBuffer([]byte{})
	out := bytes.NewBuffer([]byte{})

	for i := 0; i < 10 && !c.session.CanUpgrade(); i++ {
		// since we are not sending any specific payload to the remote end
		// or expect any incoming payload sendFrame is used here
		if err := c.sendFrame(ctx, out, in); err != nil && err != ErrReadyUpgrade {
			return err
		}

		if out.Len() > 0 {
			return errors.New("noise payload not expected from remote endpoint during handshake")
		}
	}

	if !c.session.CanUpgrade() {
		return errors.New("handshake could not finish correctly")
	}

	session, err := c.session.Upgrade()
	if err != nil {
		return nil
	}

	c.out.Reset()
	c.in.Reset()
	c.session = session
	return nil
}

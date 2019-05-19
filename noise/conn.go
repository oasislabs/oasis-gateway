package noise

import (
	"bytes"
	"context"
	"errors"
	"io"
)

// Channel represents a channel between the local endpoint
// and the remote endpoint that a Conn uses to abstract
// the underlying transport
type Channel interface {
	// Request abstracts a request into a reader which contains the
	// request, and a writer, where the response will be written
	Request(context.Context, io.Writer, io.Reader) error
}
type ChannelFunc func(context.Context, io.Writer, io.Reader) error

func (fn ChannelFunc) Request(ctx context.Context, w io.Writer, r io.Reader) error {
	return fn(ctx, w, r)
}

// Conn represents a noise connection to a remote endpoint. It abstracts
// handling of the session state. A connection is not concurrency safe,
// if that's a needed property looked into using a FixedConnPool
type Conn struct {
	channel Channel
	session *Session

	in  *bytes.Buffer
	out *bytes.Buffer
}

// Dial creates a new connection and completes the handshake with the
// remote endpoint. If the handshake fails the Dial will also be
// considered failed
func DialContext(ctx context.Context, channel Channel, props *SessionProps) (*Conn, error) {
	if !props.Initiator {
		return nil, errors.New("when dialing the connection has to initiate the handshake")
	}

	session, err := NewSession(props)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		channel: channel,
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
func (c *Conn) Request(ctx context.Context, res io.Writer, req io.Reader) error {
	return c.request(ctx, res, req)
}

func (c *Conn) request(ctx context.Context, res io.Writer, req io.Reader) error {
	if _, err := c.session.Write(c.in, req); err != nil {
		return err
	}

	err := SerializeFrame(c.out, &OutgoingFrame{
		SessionID: c.session.ID(),
		Payload:   c.in.Bytes(),
	})

	// cleanup c.in from the request bytes now that all the content
	// is in out
	c.in.Reset()

	if err != nil {
		return err
	}

	// send request whose contents are in c.out. At this point
	// c.in should be empty
	if err := c.channel.Request(ctx, c.in, c.out); err != nil {
		return err
	}

	if _, err := c.session.Read(res, c.in); err != nil {
		return err
	}

	return nil
}

// doHandshake performs the initial handshake with the remote endpoint
func (c *Conn) doHandshake(ctx context.Context) error {
	in := bytes.NewBuffer([]byte{})
	out := bytes.NewBuffer([]byte{})

	for i := 0; i < 10 && !c.session.CanUpgrade(); i++ {
		// since we are not sending any specific payload to the remote end
		// or expect any incoming payload we set the reader and writer as nil
		if err := c.request(ctx, out, in); err != nil && err != ErrReadyUpgrade {
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

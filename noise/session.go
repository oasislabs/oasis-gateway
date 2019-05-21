package noise

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/flynn/noise"
	"github.com/oasislabs/developer-gateway/rw"
)

var (
	ErrReadyUpgrade = errors.New("session is ready to upgrade")
)

// HandshakeCompletionStage defines the completion stage
// for the session's handshake
type HandshakeCompletionStage uint

const (
	// HandshakeInit the handshake still has to start or is already
	// in process
	HandshakeInit HandshakeCompletionStage = iota

	// Handshake has completed and the session can be upgraded to
	// handle application input/output
	HandshakeCompleted

	// HandshakeClosed the handshake has ended and the session
	// has been upgraded
	HandshakeClosed
)

// HandshakeHandler is the handler the session uses to set up
// the handshake with the remote endpoint
type HandshakeHandler struct {
	stage        HandshakeCompletionStage
	state        *noise.HandshakeState
	localCipher  *noise.CipherState
	remoteCipher *noise.CipherState
}

// Upgrade the HandshakeHandler to a TransportHandler so that the
// application can send/receive payloads
func (s *HandshakeHandler) Upgrade() (*TransportHandler, error) {
	if s.stage != HandshakeCompleted {
		return nil, errors.New("Handshake has not completed")
	}

	if s.localCipher == nil || s.remoteCipher == nil {
		panic("Handshake is completed but localCipher and remoteCipher are not set")
	}

	transport := &TransportHandler{
		localCipher:  s.localCipher,
		remoteCipher: s.remoteCipher,
	}

	// clear up HandshakeState to make sure it is not reused
	s.stage = HandshakeClosed
	s.localCipher = nil
	s.remoteCipher = nil

	return transport, nil
}

// CanUpgrade returns whether the Handler is in a HandshakeCompletionStage
// in which it can be upgraded to an established session
func (s *HandshakeHandler) CanUpgrade() bool {
	return s.stage == HandshakeCompleted
}

// Read reads remote input from an io.Reader, processes that
// input and writes the output (if any needs to be generated)
// to the writer
func (s *HandshakeHandler) Read(w io.Writer, r io.Reader) (int, error) {
	if s.stage == HandshakeCompleted {
		return 0, ErrReadyUpgrade
	}

	if s.stage == HandshakeClosed {
		panic("attempt to call Read on HandshakeHandler instance that has been discarded")
	}

	in := bytes.NewBuffer(make([]byte, 0, 128))
	c, err := rw.CopyWithLimit(in, r, rw.ReadLimitProps{
		Limit:        65535,
		FailOnExceed: true,
	})
	if err != nil {
		return 0, err
	}

	p, remoteCipher, localCipher, err := s.state.ReadMessage(nil, in.Bytes()[:c])
	if err != nil {
		return 0, err
	}

	n, err := w.Write(p)
	if err != nil {
		return 0, err
	}

	if localCipher != nil && remoteCipher != nil {
		if s.stage != HandshakeInit {
			panic("invalid stage state when completing Handshakestate")
		}
		s.localCipher = localCipher
		s.remoteCipher = remoteCipher
		s.stage = HandshakeCompleted
	}

	return n, nil
}

// Write reads local input for payloads that need to be sent from
// an io.Reader, processes the input and writes the output (if
// any needs to be generated) to the writer
func (s *HandshakeHandler) Write(w io.Writer, r io.Reader) (int, error) {
	if s.stage == HandshakeCompleted {
		return 0, ErrReadyUpgrade
	}

	if s.stage == HandshakeClosed {
		panic("attempt to call Read on HandshakeHandler instance that has been discarded")
	}

	out := bytes.NewBuffer(make([]byte, 0, 128))
	c, err := rw.CopyWithLimit(out, r, rw.ReadLimitProps{
		Limit:        65535,
		FailOnExceed: true,
	})
	if err != nil {
		return 0, err
	}

	p, remoteCipher, localCipher, err := s.state.WriteMessage(nil, out.Bytes()[:c])
	if err != nil {
		return 0, err
	}

	n, err := w.Write(p)
	if err != nil {
		return 0, err
	}

	if localCipher != nil && remoteCipher != nil {
		if s.stage != HandshakeInit {
			panic("invalid stage state when completing Handshakestate")
		}

		s.localCipher = localCipher
		s.remoteCipher = remoteCipher
		s.stage = HandshakeCompleted
	}

	return n, err
}

// TransportHandler is the handler the session uses to send/receive
// data once the handshake has completed
type TransportHandler struct {
	localCipher  *noise.CipherState
	remoteCipher *noise.CipherState
	ad           []byte
}

// Write is the implementation of Write for rw.WrapReadWriter
func (s *TransportHandler) Write(w io.Writer, r io.Reader) (int, error) {
	in := bytes.NewBuffer(make([]byte, 0, 128))
	n, err := rw.CopyWithLimit(in, r, rw.ReadLimitProps{
		FailOnExceed: true,
		Limit:        65535,
	})
	if err != nil {
		return 0, err
	}

	out := s.remoteCipher.Encrypt(nil, s.ad, in.Bytes()[:n])
	return w.Write(out)
}

// Read is the implementation of Read for rw.WrapReadWriter
func (s *TransportHandler) Read(w io.Writer, r io.Reader) (int, error) {
	in := bytes.NewBuffer(make([]byte, 0, 128))
	n, err := rw.CopyWithLimit(in, r, rw.ReadLimitProps{
		FailOnExceed: true,
		Limit:        65535,
	})
	if err != nil {
		return 0, err
	}

	p, err := s.localCipher.Decrypt(nil, s.ad, in.Bytes()[:n])
	if err != nil {
		return 0, err
	}
	return w.Write(p)
}

// Session holds information on the state of a particular session,
// handling the protocol messages
type Session struct {
	canUpgrade bool
	handler    rw.BiReadWriter

	reader rw.UniRead
	writer rw.UniWrite

	id [32]byte
}

// SessionProps are the properties to configure the behaviour of
// a noise session
type SessionProps struct {
	// Initiator sets the role of this Session instance for the handshake. If
	// true, this Session initiates the handshake
	Initiator bool
}

func genSessionID(id []byte) error {
	if len(id) != 32 {
		return errors.New("session ID must have 32 bytes")
	}

	n, err := rand.Reader.Read(id)
	if err != nil {
		return err
	}

	if n != 32 {
		return errors.New("failed to write 32 random bytes to session ID")
	}

	return nil
}

// NewSession creates a new noise session with the specific configuration.
// A session is not concurrency safe until the handshake has completed
func NewSession(props *SessionProps) (*Session, error) {
	pair, err := noise.DH25519.GenerateKeypair(rand.Reader)
	if err != nil {
		return nil, err
	}

	cipherSuite := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	config := noise.Config{
		CipherSuite:   cipherSuite,
		Random:        rand.Reader,
		Pattern:       noise.HandshakeXX,
		Initiator:     props.Initiator,
		StaticKeypair: pair,
	}

	state, err := noise.NewHandshakeState(config)
	if err != nil {
		return nil, err
	}

	session := &Session{
		canUpgrade: false,
		handler:    &HandshakeHandler{state: state},
	}

	session.reader = rw.UniReadFunc(session.readHandshake)
	session.writer = rw.UniWriteFunc(session.writeHandshake)

	if err := genSessionID(session.id[:]); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Session) ID() []byte {
	return s.id[:]
}

// CanUpgrade checks if the session has finished the handshake and
// can be upgraded to transport mode
func (s *Session) CanUpgrade() bool {
	return s.canUpgrade
}

// Upgrades the session after the handler has been completed
func (s *Session) Upgrade() (*Session, error) {
	if !s.canUpgrade {
		return nil, errors.New("session is not ready to be upgraded")
	}

	handler := s.handler.(*HandshakeHandler)
	if !handler.CanUpgrade() {
		panic("attempt to upgrade session when handshake has not been completed")
	}

	newHandler, err := handler.Upgrade()
	if err != nil {
		panic(fmt.Sprintf("failed to upgrade session %s", err))
	}

	newSession := &Session{
		id:         s.id,
		canUpgrade: false,
		handler:    newHandler,
	}

	newSession.reader = rw.UniReadFunc(newSession.read)
	newSession.writer = rw.UniWriteFunc(newSession.write)

	return newSession, nil
}

func (s *Session) writeHandshake(w io.Writer, r io.Reader) (int, error) {
	handler := s.handler.(*HandshakeHandler)
	n, err := handler.Write(w, r)
	if err != nil {
		return 0, err
	}

	if handler.CanUpgrade() {
		s.canUpgrade = true
	}
	return n, nil
}

func (s *Session) readHandshake(w io.Writer, r io.Reader) (int, error) {
	handler := s.handler.(*HandshakeHandler)
	n, err := handler.Read(w, r)
	if err != nil {
		return 0, err
	}

	if handler.CanUpgrade() {
		s.canUpgrade = true
	}

	return n, nil
}

func (s *Session) write(w io.Writer, r io.Reader) (int, error) {
	return s.handler.Write(w, r)
}

func (s *Session) read(w io.Writer, r io.Reader) (int, error) {
	return s.handler.Read(w, r)
}

// Write bytes to the session
func (s *Session) Write(w io.Writer, r io.Reader) (int, error) {
	return s.writer.Write(w, r)
}

// Read bytes from the session
func (s *Session) Read(w io.Writer, r io.Reader) (int, error) {
	return s.reader.Read(w, r)
}

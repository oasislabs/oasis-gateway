package noise

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/flynn/noise"
)

type HandshakeCompletionStage uint

const (
	HandshakeInit HandshakeCompletionStage = iota
	HandshakeCompleted
	HandshakeClosed
)

type WrapReadWriter interface {
	Read(io.Reader, []byte) ([]byte, int, error)
	Write(io.Writer, []byte) (int, error)
}

type HandshakeHandler struct {
	stage        HandshakeCompletionStage
	state        *noise.HandshakeState
	localCipher  *noise.CipherState
	remoteCipher *noise.CipherState
}

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

func (s *HandshakeHandler) CanUpgrade() bool {
	return s.stage == HandshakeCompleted
}

func (s *HandshakeHandler) Read(r io.Reader, p []byte) ([]byte, int, error) {
	if s.stage != HandshakeInit {
		panic("attempt to call Read on HandshakeHandler instance that has been discarded")
	}

	in := make([]byte, 1024)
	in, n, err := readWithAppendLimit(r, in, 65535)
	if err != nil {
		return nil, 0, err
	}

	p, remoteCipher, localCipher, err := s.state.ReadMessage(p, in[:n])
	if err != nil {
		return nil, 0, err
	}

	if localCipher != nil && remoteCipher != nil {
		if s.stage != HandshakeInit {
			panic("invalid stage state when completing Handshakestate")
		}
		s.localCipher = localCipher
		s.remoteCipher = remoteCipher
		s.stage = HandshakeCompleted
	}

	return p, len(p), nil
}

func (s *HandshakeHandler) Write(w io.Writer, p []byte) (int, error) {
	if s.stage != HandshakeInit {
		panic("attempt to call Output on HandshakeHandler instance that has been discarded")
	}

	out := make([]byte, 0, 1024)
	out, remoteCipher, localCipher, err := s.state.WriteMessage(out, p)
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

	return w.Write(out)
}

type TransportHandler struct {
	localCipher  *noise.CipherState
	remoteCipher *noise.CipherState
	ad           []byte
}

func (s *TransportHandler) Write(w io.Writer, p []byte) (int, error) {
	out := make([]byte, 0, 1024)
	out = s.remoteCipher.Encrypt(out, s.ad, p)
	return w.Write(out)
}

func (s *TransportHandler) Read(r io.Reader, p []byte) ([]byte, int, error) {
	in := make([]byte, 0, 1024)
	in, _, err := readWithAppendLimit(r, in, 65535)
	if err != nil {
		return nil, 0, err
	}

	p, err = s.localCipher.Decrypt(p, s.ad, in)
	return p, len(p), err
}

// Session holds information on the state of a particular session,
// handling the protocol messages
type Session struct {
	canUpgrade bool
	handler    WrapReadWriter

	readFunc  func(io.Reader, []byte) ([]byte, int, error)
	writeFunc func(io.Writer, []byte) (int, error)
}

// SessionProps are the properties to configure the behaviour of
// a noise session
type SessionProps struct {
	// Initiator sets the role of this Session instance for the handshake. If
	// true, this Session initiates the handshake
	Initiator bool
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
		Prologue:      nil,
		PresharedKey:  nil,
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

	session.readFunc = session.readHandshake
	session.writeFunc = session.writeHandshake
	return session, nil
}

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
		canUpgrade: false,
		handler:    newHandler,
	}

	newSession.readFunc = newSession.read
	newSession.writeFunc = newSession.write

	return newSession, nil
}

func (s *Session) writeHandshake(w io.Writer, p []byte) (int, error) {
	handler := s.handler.(*HandshakeHandler)
	n, err := handler.Write(w, p)
	if handler.CanUpgrade() {
		s.canUpgrade = true
	}
	return n, err
}

func (s *Session) readHandshake(r io.Reader, p []byte) ([]byte, int, error) {
	handler := s.handler.(*HandshakeHandler)
	p, n, err := handler.Read(r, p)
	if err != nil {
		return nil, 0, err
	}

	if handler.CanUpgrade() {
		s.canUpgrade = true
	}

	return p, n, err
}

func (s *Session) write(w io.Writer, p []byte) (int, error) {
	return s.handler.Write(w, p)
}

func (s *Session) read(r io.Reader, p []byte) ([]byte, int, error) {
	return s.handler.Read(r, p)
}

// Write bytes to the session
func (s *Session) Write(w io.Writer, p []byte) (int, error) {
	return s.writeFunc(w, p)
}

// Read bytes from the session
func (s *Session) Read(r io.Reader, p []byte) ([]byte, int, error) {
	return s.readFunc(r, p)
}

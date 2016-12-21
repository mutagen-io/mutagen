package rpc

import (
	"encoding/gob"
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/stream"
)

// ClientStream provides object streaming facilities (using gob encoding) for
// use with RPC clients. Its Close method follows the semantics of net.Conn's
// Close method. Specifically, it will unblock any Encode/Decode calls that are
// in-progress.
type ClientStream struct {
	*gob.Encoder
	*gob.Decoder
	io.Closer
}

// HandlerStream provides object streaming facilities (using gob encoding) for
// use with RPC handlers.
type HandlerStream struct {
	*gob.Encoder
	*gob.Decoder
}

type Client struct {
	openerLock sync.Mutex
	opener     stream.Opener
}

func NewClient(opener stream.Opener) *Client {
	return &Client{opener: opener}
}

func (c *Client) Invoke(method string) (*ClientStream, error) {
	// Open a connection.
	c.openerLock.Lock()
	connection, err := c.opener.Open()
	c.openerLock.Unlock()
	if err != nil {
		return nil, errors.Wrap(err, "unable to open connection to server")
	}

	// Create a client stream on top of the connection.
	stream := &ClientStream{
		gob.NewEncoder(connection),
		gob.NewDecoder(connection),
		connection,
	}

	// Send the invocation request.
	if err := stream.Encode(method); err != nil {
		stream.Close()
		return nil, errors.Wrap(err, "unable to send invocation request")
	}

	// Success.
	return stream, nil
}

type Handler func(*HandlerStream)

type Service interface {
	Methods() map[string]Handler
}

type Server struct {
	handlersLock sync.RWMutex
	handlers     map[string]Handler
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]Handler),
	}
}

func (s *Server) Register(service Service) {
	// Lock the handlers registry for writing and defer its release.
	s.handlersLock.Lock()
	defer s.handlersLock.Unlock()

	// Register each of the service's methods handlers map. If two services try
	// to register the same method, this is a logic error.
	for name, method := range service.Methods() {
		if _, ok := s.handlers[name]; ok {
			panic("two methods registered with the same name")
		}
		s.handlers[name] = method
	}
}

func (s *Server) serveConnection(connection net.Conn) {
	// Ensure that the connection is closed once the handler is finished.
	defer connection.Close()

	// Create a handler stream on top of the connection.
	stream := &HandlerStream{
		gob.NewEncoder(connection),
		gob.NewDecoder(connection),
	}

	// Receive the invocation request.
	var method string
	if stream.Decode(&method) != nil {
		return
	}

	// Find and invoke the handler.
	s.handlersLock.RLock()
	handler := s.handlers[method]
	s.handlersLock.RUnlock()
	if handler != nil {
		handler(stream)
	}
}

func (s *Server) Serve(acceptor stream.Acceptor) error {
	// Accept and serve connections until there is an error with the acceptor.
	for {
		connection, err := acceptor.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting connection")
		}
		go s.serveConnection(connection)
	}
}

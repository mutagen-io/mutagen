package rpc

import (
	"encoding/gob"
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/stream"
)

type ClientStream struct {
	*gob.Encoder
	*gob.Decoder
	io.Closer
}

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
	handlers map[string]Handler
}

func NewServer() *Server {
	return &Server{handlers: make(map[string]Handler)}
}

func (s *Server) Register(service Service) {
	// Grab the service's methods.
	methods := service.Methods()

	// Register each of them in the master handlers map. If two services try to
	// register the same method, this is a logic error.
	for name, method := range methods {
		// Make sure there is no existing method with this name. If two services
		// try to register the same method, this is a logic error.
		if _, ok := s.handlers[name]; ok {
			panic("two methods registered with the same name")
		}

		// Register the method.
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

	// Receive the invocation header.
	var method string
	if stream.Decode(&method) != nil {
		return
	}

	// Find and invoke the handler.
	handler := s.handlers[method]
	if handler != nil {
		handler(stream)
	}
}

func (s *Server) Serve(listener stream.Acceptor) error {
	// Accept and serve connections until there is an error with the listener.
	for {
		connection, err := listener.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting connection")
		}
		go s.serveConnection(connection)
	}
}

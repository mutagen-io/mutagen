package rpc

import (
	"encoding/gob"
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"
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

type Opener interface {
	Open() (net.Conn, error)
}

type Client struct {
	openerLock sync.Mutex
	opener     Opener
}

func NewClient(opener Opener) *Client {
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

type Server struct {
	handlers map[string]Handler
}

func NewServer(handlers map[string]Handler) *Server {
	return &Server{handlers: handlers}
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

func (s *Server) Serve(listener net.Listener) error {
	// Accept and serve connections until there is an error with the listener.
	for {
		connection, err := listener.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting connection")
		}
		go s.serveConnection(connection)
	}
}

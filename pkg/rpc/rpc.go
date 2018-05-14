package rpc

import (
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/hashicorp/yamux"
)

type Client struct {
	multiplexerLock sync.Mutex
	multiplexer     *yamux.Session
}

func NewClient(stream io.ReadWriteCloser) (*Client, error) {
	// Create the multiplexer.
	multiplexer, err := yamux.Client(stream, yamux.DefaultConfig())
	if err != nil {
		return nil, errors.Wrap(err, "unable to create multiplexer")
	}

	// Create the client.
	return &Client{multiplexer: multiplexer}, nil
}

func (c *Client) Invoke(method string) (ClientStream, error) {
	// Open a connection.
	c.multiplexerLock.Lock()
	connection, err := c.multiplexer.Open()
	c.multiplexerLock.Unlock()
	if err != nil {
		return nil, errors.Wrap(err, "unable to open connection to server")
	}

	// Create a stream on top of the connection.
	stream := newStream(connection)

	// Send the invocation request.
	if err := stream.Send(method); err != nil {
		stream.Close()
		return nil, errors.Wrap(err, "unable to send invocation request")
	}

	// Success.
	return stream, nil
}

func (c *Client) Close() error {
	// Lock the multiplexer and defer its release.
	c.multiplexerLock.Lock()
	defer c.multiplexerLock.Unlock()

	// Attempt to close the multiplexer.
	return errors.Wrap(c.multiplexer.Close(), "unable to close multiplexer")
}

type Handler func(HandlerStream) error

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

func (s *Server) serveStream(rawStream *yamux.Stream) {
	// Create a message stream on top of the raw stream. Ensure that it's closed
	// when we're done with it.
	stream := newStream(rawStream)
	defer stream.Close()

	// Receive the invocation request.
	var method string
	if stream.Receive(&method) != nil {
		return
	}

	// Find the corresponding handler.
	s.handlersLock.RLock()
	handler := s.handlers[method]
	s.handlersLock.RUnlock()
	if handler == nil {
		stream.markError(errors.New("unable to find requested handler"))
		return
	}

	// Invoke the handler. This may return an error due to an underlying stream
	// error, in which case our markError call will fail, so we just ignore that
	// case since we can't do anything about it.
	if err := handler(stream); err != nil {
		stream.markError(err)
	}
}

func (s *Server) multiplexAndServe(connection io.ReadWriteCloser) error {
	// Wrap the connection in a multiplexer and defer its closure.
	multiplexer, err := yamux.Server(connection, yamux.DefaultConfig())
	if err != nil {
		connection.Close()
		return errors.Wrap(err, "unable to create multiplexer")
	}
	defer multiplexer.Close()

	// Accept and serve streams until there is an error with the multiplexer.
	for {
		stream, err := multiplexer.AcceptStream()
		if err != nil {
			return errors.Wrap(err, "unable to accept stream")
		}
		go s.serveStream(stream)
	}
}

func (s *Server) Serve(listener net.Listener) error {
	// Defer closure of the listener.
	defer listener.Close()

	// Accept and serve until there's an error with the listener.
	for {
		connection, err := listener.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting connection")
		}
		go s.multiplexAndServe(connection)
	}
}

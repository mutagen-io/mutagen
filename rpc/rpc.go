package rpc

import (
	"net"
	"sync"

	"github.com/pkg/errors"

	streampkg "github.com/havoc-io/mutagen/stream"
)

type Client struct {
	openerLock sync.Mutex
	opener     streampkg.Opener
}

func NewClient(opener streampkg.Opener) *Client {
	return &Client{opener: opener}
}

func (c *Client) Invoke(method string) (ClientStream, error) {
	// Open a connection.
	c.openerLock.Lock()
	connection, err := c.opener.Open()
	c.openerLock.Unlock()
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

func (s *Server) serveConnection(connection net.Conn) {
	// Ensure that the connection is closed once the handler is finished.
	defer connection.Close()

	// Create a stream on top of the connection. Ensure that it's closed when
	// we're done with it.
	stream := newStream(connection)
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

func (s *Server) Serve(acceptor streampkg.Acceptor) error {
	// Accept and serve connections until there is an error with the acceptor.
	for {
		connection, err := acceptor.Accept()
		if err != nil {
			return errors.Wrap(err, "error accepting connection")
		}
		go s.serveConnection(connection)
	}
}

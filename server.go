package drops

import (
	"net"
	"sync"

	"github.com/golang/glog"
)

// Server handles accepting connections and keeping state.
// It's broken out for testing purposes.
type Server struct {
	listener        net.Listener
	maxMetricPoints int

	stations  map[string]*Station
	stationsM sync.RWMutex
}

// NewServer constructs and returns a Server.
func NewServer(listener net.Listener, maxMetricPoints int) *Server {
	return &Server{
		listener:        listener,
		maxMetricPoints: maxMetricPoints,

		stations:  map[string]*Station{},
		stationsM: sync.RWMutex{},
	}
}

// Serve is the main acceptor loop.
func (s *Server) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			glog.Errorf("couldn't accept connection: %v", err)
			continue
		}

		go s.handle(conn)
	}
}

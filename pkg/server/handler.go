package server

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type clientConn struct {
	net.Conn

	// If the TCP client has REGISTERed, this will be filled in.
	name string
}

type metric struct {
	ts    time.Time
	value float64
}

// Station holds monitoring data about a given station.
type Station struct {
	m       sync.Mutex
	metrics map[string][]metric

	c    *clientConn
	tipe string

	runs  map[string]*run
	runsM sync.Mutex
}

type run struct {
	client *clientConn
	name   string
}

type handlerFunc func(*clientConn, string, ...string) (string, error)

// REGISTER cmd
// Expected args:
//  - [name]
//  - [type]
func (s *Server) handleRegister(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) != 2 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	name, tipe := args[0], args[1]
	if _, present := s.stations[name]; present {
		return "", errors.Errorf("%s already registered", name)
	}

	s.stations[name] = &Station{
		metrics: map[string][]metric{},

		c:    conn,
		tipe: tipe,

		runs: map[string]*run{},
	}
	conn.name = name

	return "ACK", nil
}

// LIST cmd
// Expected args: none
func (s *Server) handleList(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) != 0 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	buf := bytes.NewBufferString("LIST")
	for name, s := range s.stations {
		buf.WriteString(fmt.Sprintf(" %s:%s", name, s.tipe))
	}

	return buf.String(), nil
}

// METRIC cmd
// Expected args:
//  - [name]
//  - [float]
func (s *Server) handleMetric(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) != 2 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	name, stringValue := args[0], args[1]
	floatValue, err := strconv.ParseFloat(stringValue, 64)
	if err != nil {
		return "", err
	}

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	// client must have run REGISTER first
	if conn.name == "" {
		return "", errors.Errorf("client is not a station and cannot report telemetry")
	}

	station, ok := s.stations[conn.name]
	if !ok {
		return "", errors.Errorf("station %s is somehow unknown to us", conn.name)
	}

	station.m.Lock()
	defer station.m.Unlock()

	station.metrics[name] = append(station.metrics[name], metric{ts: s.Clock.Now(), value: floatValue})
	// to conserve memory just a bit we only keep a certain number of metrics around.
	if len(station.metrics[name]) > s.maxMetricPoints {
		_, station.metrics[name] = station.metrics[name][0], station.metrics[name][1:]
	}

	return "ACK", nil
}

// METRICS cmd
// Expected arguments:
//  - [name]
//  - [metric] (optional)
func (s *Server) handleMetrics(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) < 1 || len(args) > 2 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	name := args[0]

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	station, ok := s.stations[name]
	if !ok {
		return "", errors.Errorf("station %s is somehow unknown to us", name)
	}

	station.m.Lock()
	defer station.m.Unlock()

	buf := bytes.NewBufferString(fmt.Sprintf("METRICS %s", name))

	switch len(args) {
	case 1:
		// METRICS [name] only lists the available metrics.
		for name := range station.metrics {
			buf.WriteString(fmt.Sprintf(" %s", name))
		}
	case 2:
		// METRICS [name] [metric] lists all known values for the metric.
		metric := args[1]
		ms, ok := station.metrics[metric]
		if !ok {
			return "", errors.Errorf("no known metric %s on station %s", metric, name)
		}

		buf.WriteString(fmt.Sprintf(" %s", metric))
		for _, m := range ms {
			buf.WriteString(fmt.Sprintf(" %d:%.2f", m.ts.Unix(), m.value))
		}
	}

	return buf.String(), nil
}

// RUN cmd
// Expected arguments:
//  - [name]
//  - [function]
//  - [parameter] (optional)
func (s *Server) handleRun(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) < 2 || len(args) > 3 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	name, fn := args[0], args[1]

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	station, ok := s.stations[name]
	if !ok {
		return "", errors.Errorf("station %s is somehow unknown to us", name)
	}

	station.runsM.Lock()
	defer station.runsM.Unlock()

	if _, ok := station.runs[uid]; ok {
		return "", errors.Errorf("uid %s already in use", uid)
	}

	// route the command to the proper station connection
	fmt.Fprintf(station.c, "%s RUN %s", uid, fn)

	if len(args) == 3 {
		// include the parameter if the client specified it
		fmt.Fprintf(station.c, " %s", args[2])
	}

	// always include the needed newline
	fmt.Fprintf(station.c, "\n")

	// save the client connection so we can route back to it later.
	station.runs[uid] = &run{
		client: conn,
		name:   name,
	}

	return "ACK", nil
}

// DONE cmd
// Expected arguments:
//  - [result] (optional)
func (s *Server) handleDone(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) > 1 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	// client must have run REGISTER first
	if conn.name == "" {
		return "", errors.Errorf("client is not a station and cannot respond to RPCs")
	}

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	station, ok := s.stations[conn.name]
	if !ok {
		return "", errors.Errorf("station %s is somehow unknown to us", conn.name)
	}

	station.runsM.Lock()
	defer station.runsM.Unlock()

	c, ok := station.runs[uid]
	if !ok {
		return "", errors.Errorf("unknown uid %s", uid)
	}

	// route the command to the proper client connection
	fmt.Fprintf(c.client, "%s DONE", uid)
	if len(args) == 1 {
		// include the parameter if the station specified it
		fmt.Fprintf(c.client, " %s", args[0])
	}

	// always make sure we include the newline
	fmt.Fprintf(c.client, "\n")
	delete(station.runs, uid)

	return "ACK", nil
}

// ERR cmd
// Expected arguments:
func (s *Server) handleError(conn *clientConn, uid string, args ...string) (string, error) {
	if len(args) != 0 {
		return "", errors.Errorf("bad arg count: %v", args)
	}

	// client must have run REGISTER first
	if conn.name == "" {
		return "", errors.Errorf("client is not a station and cannot respond to RPCs")
	}

	s.stationsM.Lock()
	defer s.stationsM.Unlock()

	station, ok := s.stations[conn.name]
	if !ok {
		return "", errors.Errorf("station %s is somehow unknown to us", conn.name)
	}

	station.runsM.Lock()
	defer station.runsM.Unlock()

	c, ok := station.runs[uid]
	if !ok {
		return "", errors.Errorf("unknown uid %s", uid)
	}

	// route the command to the proper client connection
	fmt.Fprintf(c.client, "%s ERR\n", uid)
	delete(station.runs, uid)

	return "ACK", nil
}

// handle performs the actual line protocol client management.
func (s *Server) handle(c net.Conn) {

	// Wrap the net.Conn so we can tag more information on it.
	conn := clientConn{
		Conn: c,
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		scan := scanner.Text()
		cmdParts := strings.Split(scan, " ")

		var fn handlerFunc

		if len(cmdParts) < 2 {
			glog.Errorf("bad line received: %s", scan)
			conn.Write([]byte("FATAL\n"))
			continue
		}

		uid, cmdName := cmdParts[0], cmdParts[1]
		switch cmdName {
		case "LIST":
			fn = s.handleList
		case "REGISTER":
			fn = s.handleRegister
		case "METRIC":
			fn = s.handleMetric
		case "METRICS":
			fn = s.handleMetrics
		case "RUN":
			fn = s.handleRun
		case "DONE":
			fn = s.handleDone
		case "ERR":
			fn = s.handleError
		default:
			glog.Errorf("no command %s known", cmdName)
			conn.Write([]byte(fmt.Sprintf("%s ERR UNRECOGNIZED CMD\n", uid)))
			continue
		}

		resp, err := fn(&conn, uid, cmdParts[2:]...)
		if err != nil {
			glog.Errorf("error processing %s: %v", cmdName, err)
			conn.Write([]byte(fmt.Sprintf("%s ERR\n", uid)))
			continue
		}

		fmt.Fprintln(conn, fmt.Sprintf("%s %s", uid, resp))
	}
	if err := scanner.Err(); err != nil {
		glog.Errorf("reading standard input: %v", err)
	}

	// Disconnected registered connections need to be removed from the list
	// of registered s.stations.
	if conn.name != "" {
		s.stationsM.Lock()
		defer s.stationsM.Unlock()

		if _, ok := s.stations[conn.name]; ok {
			delete(s.stations, conn.name)
		}

		glog.Infof("Client %s disconnected.", conn.name)

		// TODO(silversupreme): alert somehow?
	}
}

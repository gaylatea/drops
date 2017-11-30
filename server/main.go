package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/golang/glog"
)

var (
	listenAddr = flag.String("listenAddr", ":19406", "TCP address to listen on")
	maxMetrics = flag.Int("maxMetrics", 100, "max metric data points to keep for each metric from each station")

	// ssl options
	caCert  = flag.String("caCert", "ca.crt", "Only clients signed with this CA will be accepted")
	sslCert = flag.String("sslCert", "server.crt", "SSL certificate to present to clients")
	sslKey  = flag.String("sslKey", "server.key", "SSL private key to load")

	// commands that can be acted upon by the server.
	recognizedCmds = map[string]func(conn *clientConn, args ...string) (string, error){
		"LIST":     handleList,
		"REGISTER": handleRegister,

		"METRIC":  handleMetric,
		"METRICS": handleMetrics,

		"RUN":  handleRun,
		"DONE": handleDone,
		"ERR":  handleError,
	}
)

type clientConn struct {
	net.Conn

	// If the TCP client has REGISTERed, this will be filled in.
	name string
}

func init() {
	flag.Set("alsologtostderr", "true")
}

// handle performs the actual line protocol client management.
func handle(c net.Conn) {

	// Wrap the net.Conn so we can tag more information on it.
	conn := clientConn{
		Conn: c,
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		s := scanner.Text()
		cmdParts := strings.Split(s, " ")
		if len(cmdParts) < 1 {
			glog.Errorf("given command %s not actionable", s)
			conn.Write([]byte("ERR\n"))
			continue
		}

		cmdName, cmdParts := cmdParts[0], cmdParts[1:]
		if fn, ok := recognizedCmds[cmdName]; ok {
			resp, err := fn(&conn, cmdParts...)
			if err != nil {
				glog.Errorf("error processing %s: %v", cmdName, err)
				conn.Write([]byte("ERR\n"))
				continue
			}

			fmt.Fprintln(conn, resp)
		} else {
			glog.Errorf("no command %s known", cmdName)
			conn.Write([]byte("ERR UNRECOGNIZED CMD\n"))
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		glog.Errorf("reading standard input: %v", err)
	}

	// Disconnected registered connections need to be removed from the list
	// of registered stations.
	if conn.name != "" {
		stationsM.Lock()
		defer stationsM.Unlock()

		if _, ok := stations[conn.name]; ok {
			delete(stations, conn.name)
		}

		glog.Infof("Client %s disconnected.", conn.name)

		// TODO(silversupreme): alert somehow?
	}
}

func main() {
	flag.Parse()

	// setup the ssl socket
	// Load the certificates from disk
	certificate, err := tls.LoadX509KeyPair(*sslCert, *sslKey)
	if err != nil {
		glog.Fatalf("could not load server key pair: %s", err)
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(*caCert)
	if err != nil {
		glog.Fatalf("could not read ca certificate: %s", err)
	}

	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		glog.Fatalf("failed to append client certs")
	}

	// Create the TLS credentials
	creds := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		Certificates:             []tls.Certificate{certificate},
		ClientCAs:                certPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	ln, err := tls.Listen("tcp", *listenAddr, creds)
	if err != nil {
		glog.Fatalf("couldn't listen on %s: %v", *listenAddr, err)
	}

	glog.Infof("Starting SSL server on %s.", *listenAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			glog.Errorf("couldn't accept connection: %v", err)
			continue
		}

		go handle(conn)
	}
}

package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
)

var (
	addr = flag.String("addr", "localhost:19406", "drops server to connect to")

	// ssl options
	caCert  = flag.String("caCert", "ca.crt", "Only clients signed with this CA will be accepted")
	sslCert = flag.String("sslCert", "server.crt", "SSL certificate to present to clients")
	sslKey  = flag.String("sslKey", "server.key", "SSL private key to load")
)

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
		RootCAs:                  certPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", *addr, creds)
	if err != nil {
		glog.Fatalf("couldn't connect to the drops server: %v", err)
	}
	defer conn.Close()

	stdinReader := bufio.NewReader(os.Stdin)
	connReader := bufio.NewReader(conn)

	go func() {
		for {
			output, err := connReader.ReadString('\n')
			if err != nil {
				glog.Fatalf("couldn't read from conn: %v", err)
			}

			// this very complicated string here gives us a sane interaction
			// REPL pattern while still allowing us to asynchronously
			// receive information from the server and have it displayed.
			//
			// it's still a work in progress, since it needs to adequately
			// preserve the already-typed text from the user.
			os.Stdout.Write([]byte("\r\n\033[1A\r\033[1;32m< " + output + "\033[0m> "))
		}
	}()

	// TODO(silversupreme): lock the display if the user is typing
	// so that async messages received from the server don't overwrite
	// the display the user is seeing and confusing them.

	for {
		fmt.Printf("> ")

		// interactive REPL for drops commands
		output, err := stdinReader.ReadString('\n')
		if err != nil {
			glog.Fatalf("couldn't read from conn: %v", err)
		}

		fmt.Fprintf(conn, output)
	}
}

package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/silversupreme/drops"
)

var (
	listenAddr = flag.String("listenAddr", ":19406", "TCP address to listen on")
	maxMetrics = flag.Int("maxMetrics", 100, "max metric data points to keep for each metric from each station")

	// ssl options
	caCert  = flag.String("caCert", "ca.crt", "Only clients signed with this CA will be accepted")
	sslCert = flag.String("sslCert", "server.crt", "SSL certificate to present to clients")
	sslKey  = flag.String("sslKey", "server.key", "SSL private key to load")
)

func init() {
	flag.Set("alsologtostderr", "true")
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
	s := drops.NewServer(ln, *maxMetrics)
	s.Serve()
}

package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/mwitkow/grpc-proxy/proxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	flagBackendHostPort = pflag.String(
		"backend_addr",
		"",
		"A host:port (IP or hostname) of the gRPC server to forward it to.")

	flagBackendIsUsingTls = pflag.Bool(
		"backend_tls",
		false,
		"Whether the gRPC server of the backend is serving in plaintext (false) or over TLS (true).",
	)

	flagBackendTlsNoVerify = pflag.Bool(
		"backend_tls_noverify",
		false,
		"Whether to ignore TLS verification checks (cert validity, hostname). *DO NOT USE IN PRODUCTION*.",
	)

	flagMaxCallRecvMsgSize = pflag.Int(
		"backend_max_call_recv_msg_size",
		1024*1024*4, // The current maximum receive msg size per https://github.com/grpc/grpc-go/blob/v1.8.2/server.go#L54
		"Maximum receive message size limit. If not specified, the default of 4MB will be used.",
	)

	flagBackendTlsCa = pflag.StringSlice(
		"backend_tls_ca_files",
		[]string{},
		"Paths (comma separated) to PEM certificate chains used for verification of backend certificates. If empty, host CA chain will be used.",
	)
)

func dialBackendOrFail() *grpc.ClientConn {
	if *flagBackendHostPort == "" {
		logrus.Fatalf("flag 'backend_addr' must be set")
	}
	opt := []grpc.DialOption{}
	opt = append(opt, grpc.WithCodec(proxy.Codec()))
	if *flagBackendIsUsingTls {
		opt = append(opt, grpc.WithTransportCredentials(credentials.NewTLS(buildBackendTlsOrFail())))
	} else {
		opt = append(opt, grpc.WithInsecure())
	}
	opt = append(opt, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(*flagMaxCallRecvMsgSize)))
	cc, err := grpc.Dial(*flagBackendHostPort, opt...)
	if err != nil {
		logrus.Fatalf("failed dialing backend: %v", err)
	}
	return cc
}

func buildBackendTlsOrFail() *tls.Config {
	tlsConfig := &tls.Config{}
	tlsConfig.MinVersion = tls.VersionTLS12
	if *flagBackendTlsNoVerify {
		tlsConfig.InsecureSkipVerify = true
	} else {
		if len(*flagBackendTlsCa) > 0 {
			tlsConfig.RootCAs = x509.NewCertPool()
			for _, path := range *flagBackendTlsCa {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					logrus.Fatalf("failed reading backend CA file %v: %v", path, err)
				}
				if ok := tlsConfig.RootCAs.AppendCertsFromPEM(data); !ok {
					logrus.Fatalf("failed processing backend CA file %v", path)
				}
			}
		}
	}
	return tlsConfig
}

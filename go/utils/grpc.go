// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/juju/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"io/ioutil"
	"time"
)

type ServerConfig struct {
	ListenAddr string
	PathCrt    string
	PathKey    string
}

func (srv *ServerConfig) ServeTLS() (*grpc.Server, error) {
	if len(srv.PathCrt) <= 0 {
		return nil, errors.NotValidf("invalid TLS/x509 certificate path [%s]", srv.PathCrt)
	}
	if len(srv.PathKey) <= 0 {
		return nil, errors.NotValidf("invalid TLS/x509 key path [%s]", srv.PathKey)
	}
	var certBytes, keyBytes []byte
	var err error

	Logger.Info().Str("key", srv.PathKey).Str("crt", srv.PathCrt).Msg("TLS config")

	if certBytes, err = ioutil.ReadFile(srv.PathCrt); err != nil {
		return nil, errors.Annotate(err, "certificate file error")
	}
	if keyBytes, err = ioutil.ReadFile(srv.PathKey); err != nil {
		return nil, errors.Annotate(err, "key file error")
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(certBytes)
	if !ok {
		return nil, errors.New("invalid certificates")
	}

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, errors.Annotate(err, "x509 key pair error")
	}

	return grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.KeepaliveParams(keepaliveServer),
		grpc.KeepaliveEnforcementPolicy(keepalivePolicy),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			NewUnaryServerInterceptorZerolog())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			NewStreamServerInterceptorZerolog()))), nil
}

// keepaliveServer lets the hub notice an agent that has gone away without
// saying so, which is the mirror of the client-side problem: a control stream
// is idle most of the time, so nothing else would reveal it.
var keepaliveServer = keepalive.ServerParameters{
	Time:    30 * time.Second,
	Timeout: 10 * time.Second,
}

// keepalivePolicy has to admit the pings the agents send. The gRPC default
// refuses a ping from a client with no active stream and answers with GOAWAY,
// which would turn the client keepalive above into a reconnect loop.
var keepalivePolicy = keepalive.EnforcementPolicy{
	MinTime:             15 * time.Second,
	PermitWithoutStream: true,
}

func (srv *ServerConfig) ServeInsecure() (*grpc.Server, error) {
	return grpc.NewServer(
		grpc.KeepaliveParams(keepaliveServer),
		grpc.KeepaliveEnforcementPolicy(keepalivePolicy),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			NewUnaryServerInterceptorZerolog())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			NewStreamServerInterceptorZerolog()))), nil
}

func DialTLS(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	return nil, errors.NotImplemented
}

// keepaliveClient makes gRPC prove the peer is still there.
//
// This is not a nicety. The upstream agent's whole reconnect logic hangs off
// stream.Recv returning an error, and the hub sends nothing on the control
// stream unless a viewer asks for a camera -- so an idle stream carries no
// traffic at all. Without a keepalive, a NAT between an agent and the hub drops
// the connection after its idle timeout, nobody is told, Recv parks forever,
// and the agent is a zombie that still reports its link as up.
//
// PermitWithoutStream is on because the interesting case is exactly the one
// where no RPC is in flight.
var keepaliveClient = keepalive.ClientParameters{
	Time:                30 * time.Second,
	Timeout:             10 * time.Second,
	PermitWithoutStream: true,
}

func DialInsecure(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepaliveClient),
	)
}

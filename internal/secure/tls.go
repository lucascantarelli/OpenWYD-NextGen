// Package secure monta as credenciais de transporte gRPC dos links internos
// entre serviços (tm↔db, tm↔bin). Esses links são modernos e controlados nas duas
// pontas, então rodam mTLS — diferente da borda do cliente, presa ao CPSock
// legado.
//
// O material de certificado é referenciado só por caminho de arquivo (vindo de
// flag/ambiente); nada é embutido. Sem caminhos configurados, as credenciais
// caem em transporte inseguro para o bring-up local — produção precisa definir
// os três caminhos.
package secure

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Config aponta para os arquivos PEM de um lado do link mTLS.
type Config struct {
	CertFile   string // certificado deste par
	KeyFile    string // chave privada deste par
	CAFile     string // CA que assina o certificado do outro par
	ServerName string // nome esperado do servidor (só no lado cliente)
}

// Enabled diz se algum material TLS foi configurado. Quando falso, os helpers de
// credencial devolvem transporte inseguro.
func (c Config) Enabled() bool {
	return c.CertFile != "" || c.KeyFile != "" || c.CAFile != ""
}

// ServerCreds monta as credenciais de um servidor gRPC. Com TLS ligado, exige e
// confere o certificado do cliente (TLS mútuo).
func ServerCreds(c Config) (credentials.TransportCredentials, error) {
	if !c.Enabled() {
		return insecure.NewCredentials(), nil
	}
	cert, pool, err := load(c)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}), nil
}

// ClientCreds monta as credenciais de um cliente gRPC, apresentando o
// certificado deste par e conferindo o servidor contra a CA configurada.
func ClientCreds(c Config) (credentials.TransportCredentials, error) {
	if !c.Enabled() {
		return insecure.NewCredentials(), nil
	}
	cert, pool, err := load(c)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   c.ServerName,
		MinVersion:   tls.VersionTLS13,
	}), nil
}

// load lê o par de chaves e o pool de CAs, exigindo os três caminhos.
func load(c Config) (tls.Certificate, *x509.CertPool, error) {
	if c.CertFile == "" || c.KeyFile == "" || c.CAFile == "" {
		return tls.Certificate{}, nil, fmt.Errorf("secure: cert, key and CA files are all required for mTLS")
	}
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("secure: load keypair: %w", err)
	}
	caPEM, err := os.ReadFile(c.CAFile)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("secure: read CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return tls.Certificate{}, nil, fmt.Errorf("secure: no certificates parsed from %s", c.CAFile)
	}
	return cert, pool, nil
}

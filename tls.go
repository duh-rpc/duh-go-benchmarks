package benchmark

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"strings"
	"time"
)

const (
	blockTypeEC   = "EC PRIVATE KEY"
	blockTypeCert = "CERTIFICATE"
)

type TLSConfig struct {
	// (Optional) Sets the Client Authentication type as defined in the 'tls' package.
	// Defaults to tls.NoClientCert.See the standard library tls.ClientAuthType for valid values.
	// If set to anything but tls.NoClientCert then SetupTLS() attempts to load ClientAuthCaFile,
	// ClientAuthKeyFile and ClientAuthCertFile and sets those certs into the ClientTLS struct. If
	// none of the ClientXXXFile's are set, uses KeyFile and CertFile for client authentication.
	ClientAuth tls.ClientAuthType

	// (Optional) The path to the Trusted Certificate Authority used for client auth. If ClientAuth is
	// set and this field is empty, then CaFile is used to auth clients.
	ClientAuthCaFile string

	// (Optional) The path to the client private key, which is used to create the ClientTLS config. If
	// ClientAuth is set and this field is empty then KeyFile is used to create the ClientTLS.
	ClientAuthKeyFile string

	// (Optional) The path to the client cert key, which is used to create the ClientTLS config. If
	// ClientAuth is set and this field is empty then KeyFile is used to create the ClientTLS.
	ClientAuthCertFile string

	// (Optional) If InsecureSkipVerify is true, TLS clients will accept any certificate
	// presented by the server and any host name in that certificate.
	InsecureSkipVerify bool

	// (Optional) The CA Certificate in PEM format. Used if CaFile is unset
	CaPEM *bytes.Buffer

	// (Optional) The CA Private Key in PEM format. Used if CaKeyFile is unset
	CaKeyPEM *bytes.Buffer

	// (Optional) The Certificate Key in PEM format. Used if KeyFile is unset.
	KeyPEM *bytes.Buffer

	// (Optional) The Certificate in PEM format. Used if CertFile is unset.
	CertPEM *bytes.Buffer

	// (Optional) The client auth CA Certificate in PEM format. Used if ClientAuthCaFile is unset.
	ClientAuthCaPEM *bytes.Buffer

	// (Optional) The client auth private key in PEM format. Used if ClientAuthKeyFile is unset.
	ClientAuthKeyPEM *bytes.Buffer

	// (Optional) The client auth Certificate in PEM format. Used if ClientAuthCertFile is unset.
	ClientAuthCertPEM *bytes.Buffer

	// (Optional) the server name to check when validating the provided certificate
	ClientAuthServerName string

	// (Optional) The config created for use by the gubernator server. If set, all other
	// fields in this struct are ignored and this config is used. If unset, gubernator.SetupTLS()
	// will create a config using the above fields.
	ServerTLS *tls.Config

	// (Optional) The config created for use by gubernator clients and peer communication. If set, all other
	// fields in this struct are ignored and this config is used. If unset, gubernator.SetupTLS()
	// will create a config using the above fields.
	ClientTLS *tls.Config
}

func SetupTLS(conf *TLSConfig) error {
	if conf == nil {
		return nil
	}

	// Basic config with reasonably secure defaults
	//setter.SetDefault(&conf.ServerTLS, &tls.Config{
	//	CipherSuites: []uint16{
	//		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	//		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	//		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	//		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	//		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	//		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	//		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	//		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	//		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	//		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	//		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	//		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	//		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	//		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	//		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	//		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	//		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	//	},
	//	ClientAuth: conf.ClientAuth,
	//	MinVersion: tls.VersionTLS13,
	//	NextProtos: []string{
	//		"h2", "http/1.1", // enable HTTP/2
	//	},
	//})
	//setter.SetDefault(&conf.ClientTLS, &tls.Config{})

	conf.ServerTLS = &tls.Config{}
	conf.ClientTLS = &tls.Config{}

	// Generate CA Cert and Private Key
	if err := selfCA(conf); err != nil {
		return fmt.Errorf("while generating self signed CA certs: %w", err)
	}

	// Generate Server Cert and Private Key
	if err := selfCert(conf); err != nil {
		return fmt.Errorf("while generating self signed server certs: %w", err)
	}

	if conf.CaPEM != nil {
		rootPool, err := x509.SystemCertPool()
		if err != nil {
			log.Printf("while loading system CA Certs '%s'; using provided pool instead", err)
			rootPool = x509.NewCertPool()
		}
		rootPool.AppendCertsFromPEM(conf.CaPEM.Bytes())
		conf.ServerTLS.RootCAs = rootPool
		conf.ClientTLS.RootCAs = rootPool
	}

	if conf.KeyPEM != nil && conf.CertPEM != nil {
		serverCert, err := tls.X509KeyPair(conf.CertPEM.Bytes(), conf.KeyPEM.Bytes())
		if err != nil {
			return fmt.Errorf("while parsing server certificate and private key: %w", err)
		}
		conf.ServerTLS.Certificates = []tls.Certificate{serverCert}
		conf.ClientTLS.Certificates = []tls.Certificate{serverCert}
	}

	// If user asked for client auth
	if conf.ClientAuth != tls.NoClientCert {
		clientPool := x509.NewCertPool()
		if conf.ClientAuthCaPEM != nil {
			// If client auth CA was provided
			clientPool.AppendCertsFromPEM(conf.ClientAuthCaPEM.Bytes())

		} else if conf.CaPEM != nil {
			// else use the servers CA
			clientPool.AppendCertsFromPEM(conf.CaPEM.Bytes())
		}

		// tlsCert.RootCAs.Subjects is deprecated because cert does not come from SystemCertPool.
		if len(clientPool.Subjects()) == 0 {
			return errors.New("client auth enabled, but no CA's provided")
		}

		conf.ServerTLS.ClientCAs = clientPool

		// If client auth key/cert was provided
		if conf.ClientAuthKeyPEM != nil && conf.ClientAuthCertPEM != nil {
			clientCert, err := tls.X509KeyPair(conf.ClientAuthCertPEM.Bytes(), conf.ClientAuthKeyPEM.Bytes())
			if err != nil {
				return fmt.Errorf("while parsing client certificate and private key: %w", err)
			}
			conf.ClientTLS.Certificates = []tls.Certificate{clientCert}
		}
	}

	conf.ClientTLS.ServerName = conf.ClientAuthServerName
	conf.ClientTLS.InsecureSkipVerify = conf.InsecureSkipVerify
	return nil
}

func selfCert(conf *TLSConfig) error {
	if conf.CertPEM != nil && conf.KeyPEM != nil {
		return nil
	}

	cert := x509.Certificate{
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{Organization: []string{"gubernator"}},
		NotAfter:              time.Now().Add(365 * (24 * time.Hour)),
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		SerialNumber:          big.NewInt(0xC0FFEE),
		NotBefore:             time.Now(),
		BasicConstraintsValid: true,
	}

	log.Print("Generating Server Private Key and Certificate....")
	log.Printf("Cert DNS names: (%s)", strings.Join(cert.DNSNames, ","))
	log.Printf("Cert IPs: (%s)", func() string {
		var r []string
		for i := range cert.IPAddresses {
			r = append(r, cert.IPAddresses[i].String())
		}
		return strings.Join(r, ",")
	}())

	// Generate a public / private key
	privKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return fmt.Errorf("while generating pubic/private key pair: %w", err)
	}

	// Attempt to sign the generated certs with the provided CaFile
	if conf.CaPEM == nil && conf.CaKeyPEM == nil {
		return errors.New("unable to generate server certs without a signing CA")
	}

	keyPair, err := tls.X509KeyPair(conf.CaPEM.Bytes(), conf.CaKeyPEM.Bytes())
	if err != nil {
		return fmt.Errorf("while reading generated PEMs")
	}

	if len(keyPair.Certificate) == 0 {
		return errors.New("no certificates found in CA PEM")
	}

	caCert, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return fmt.Errorf("while parsing CA Cert")
	}

	signedBytes, err := x509.CreateCertificate(rand.Reader, &cert, caCert, &privKey.PublicKey, keyPair.PrivateKey)
	if err != nil {
		return fmt.Errorf("while self signing server cert: %w", err)
	}

	conf.CertPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.CertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: signedBytes,
	}); err != nil {
		return fmt.Errorf("while encoding CERTIFICATE PEM: %w", err)
	}

	b, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("while encoding EC Marshalling: %w", err)
	}

	conf.KeyPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.KeyPEM, &pem.Block{
		Type:  blockTypeEC,
		Bytes: b,
	}); err != nil {
		return fmt.Errorf("while encoding EC KEY PEM: %w", err)
	}
	return nil
}

func selfCA(conf *TLSConfig) error {
	ca := x509.Certificate{
		SerialNumber:          big.NewInt(2319),
		Subject:               pkix.Name{Organization: []string{"gubernator"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	var privKey *ecdsa.PrivateKey
	var err error
	var b []byte

	if conf.CaPEM != nil && conf.CaKeyPEM != nil {
		return nil
	}

	log.Print("Generating CA Certificates....")
	privKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return fmt.Errorf("while generating pubic/private key pair: %w", err)
	}

	b, err = x509.CreateCertificate(rand.Reader, &ca, &ca, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("while self signing CA certificate: %w", err)
	}

	conf.CaPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.CaPEM, &pem.Block{
		Type:  blockTypeCert,
		Bytes: b,
	}); err != nil {
		return fmt.Errorf("while encoding CERTIFICATE PEM: %w", err)
	}

	b, err = x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("while marshalling EC private key: %w", err)
	}

	conf.CaKeyPEM = new(bytes.Buffer)
	if err := pem.Encode(conf.CaKeyPEM, &pem.Block{
		Type:  blockTypeEC,
		Bytes: b,
	}); err != nil {
		return fmt.Errorf("while encoding EC private key into PEM: %w", err)
	}
	return nil
}

// Package internaltls issues the private CA and server certificates that
// encrypt traffic between Roomies containers (web -> backend, backend/worker
// -> PostgreSQL). It runs as the one-shot `tls-init` subcommand before the
// other services start. The CA is kept across runs (its key in a root-only
// directory) so reissuing one server certificate never changes the trust
// root that running services already loaded.
package internaltls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CAFile         = "ca.crt"
	CAKeyFile      = "private/ca.key"
	DBCertFile     = "db.crt"
	DBKeyFile      = "db.key"
	BackendCert    = "backend.crt"
	BackendKey     = "backend.key"
	PGHBAFile      = "pg_hba.conf"
	validity       = 10 * 365 * 24 * time.Hour
	renewThreshold = 30 * 24 * time.Hour
)

// Options names the hosts each certificate is valid for and who may read the
// private keys. Ownership is applied only when running as root.
type Options struct {
	DBHosts      []string
	BackendHosts []string
	// DBGroup is the PostgreSQL server's group id. db.key is root-owned with
	// mode 0640 so the server can read it and PostgreSQL accepts it.
	DBGroup int
	// BackendUser is the uid of the backend image's user; backend.key is 0600.
	BackendUser int
	Now         func() time.Time
}

// PGHBA rejects every non-TLS TCP connection; the image's socket stays trusted
// for initdb and pg_isready.
const PGHBA = `# Written by roomies tls-init. TCP connections must use TLS.
local     all all             trust
hostssl   all all 0.0.0.0/0   scram-sha-256
hostssl   all all ::/0        scram-sha-256
hostnossl all all 0.0.0.0/0   reject
hostnossl all all ::/0        reject
`

// Ensure writes the CA, the PostgreSQL and backend certificates, and
// pg_hba.conf into dir. A server certificate is reissued only when it is
// missing, no longer matches opts, chains to another CA, or is close to
// expiry; the CA is replaced only when it is missing or close to expiry.
func Ensure(dir string, opts Options) (issued bool, err error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if len(opts.DBHosts) == 0 || len(opts.BackendHosts) == 0 {
		return false, errors.New("db and backend hosts are required")
	}
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(CAKeyFile)), 0o700); err != nil {
		return false, err
	}
	if err := os.Chmod(filepath.Join(dir, filepath.Dir(CAKeyFile)), 0o700); err != nil {
		return false, err
	}
	now := opts.Now()
	caCert, caKey := loadCA(dir, now)
	newCA := caCert == nil
	if newCA {
		if caCert, caKey, err = issueCA(dir, now); err != nil {
			return false, err
		}
		issued = true
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	leaves := []struct {
		cert, key  string
		hosts      []string
		uid, gid   int
		keyMode    os.FileMode
		commonName string
	}{
		{DBCertFile, DBKeyFile, opts.DBHosts, 0, opts.DBGroup, 0o640, "roomies-db"},
		{BackendCert, BackendKey, opts.BackendHosts, opts.BackendUser, opts.BackendUser, 0o600, "roomies-backend"},
	}
	for _, leaf := range leaves {
		if !newCA && leafCurrent(dir, leaf.cert, leaf.key, leaf.hosts, roots, now) {
			continue
		}
		certPEM, keyPEM, err := issueLeaf(caCert, caKey, leaf.commonName, leaf.hosts, now)
		if err != nil {
			return false, err
		}
		if err := writeFile(filepath.Join(dir, leaf.key), keyPEM, leaf.keyMode, leaf.uid, leaf.gid); err != nil {
			return false, err
		}
		if err := writeFile(filepath.Join(dir, leaf.cert), certPEM, 0o644, -1, -1); err != nil {
			return false, err
		}
		issued = true
	}
	return issued, writeFile(filepath.Join(dir, PGHBAFile), []byte(PGHBA), 0o644, -1, -1)
}

// loadCA returns the stored CA when its certificate and key match and it is
// not close to expiry.
func loadCA(dir string, now time.Time) (*x509.Certificate, *ecdsa.PrivateKey) {
	cert, err := readCert(filepath.Join(dir, CAFile))
	if err != nil || !cert.IsCA || now.Add(renewThreshold).After(cert.NotAfter) {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(dir, CAKeyFile))
	if err != nil {
		return nil, nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || !key.PublicKey.Equal(cert.PublicKey) {
		return nil, nil
	}
	return cert, key
}

func issueCA(dir string, now time.Time) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "Roomies internal CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	// Root-only: other containers mount this volume but must not sign.
	if err := writeFile(filepath.Join(dir, CAKeyFile), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600, 0, 0); err != nil {
		return nil, nil, err
	}
	if err := writeFile(filepath.Join(dir, CAFile), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644, -1, -1); err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func issueLeaf(ca *x509.Certificate, caKey *ecdsa.PrivateKey, commonName string, hosts []string, now time.Time) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(validity - time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, host)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
}

// leafCurrent reports a server certificate whose key exists, that chains to
// roots for every host, and that is not close to expiry.
func leafCurrent(dir, certFile, keyFile string, hosts []string, roots *x509.CertPool, now time.Time) bool {
	if _, err := os.Stat(filepath.Join(dir, keyFile)); err != nil {
		return false
	}
	cert, err := readCert(filepath.Join(dir, certFile))
	if err != nil || now.Add(renewThreshold).After(cert.NotAfter) {
		return false
	}
	for _, host := range hosts {
		if _, err := cert.Verify(x509.VerifyOptions{DNSName: host, Roots: roots, CurrentTime: now}); err != nil {
			return false
		}
	}
	return true
}

func readCert(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("%s is not a PEM certificate", path)
	}
	return x509.ParseCertificate(block.Bytes)
}

// writeFile replaces path atomically so a reader never sees a partial key.
func writeFile(path string, data []byte, mode os.FileMode, uid, gid int) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if os.Geteuid() == 0 && (uid >= 0 || gid >= 0) {
		if err := os.Chown(tmp.Name(), uid, gid); err != nil {
			return err
		}
	}
	return os.Rename(tmp.Name(), path)
}

func serial() *big.Int {
	n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		panic(err)
	}
	return n
}

// SplitHosts parses a comma-separated host list.
func SplitHosts(value string) []string {
	var hosts []string
	for _, host := range strings.Split(value, ",") {
		if host = strings.TrimSpace(host); host != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

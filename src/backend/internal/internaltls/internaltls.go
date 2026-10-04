// Package internaltls issues the private CA and server certificates that
// encrypt traffic between Roomies containers (web -> backend, backend/worker
// -> PostgreSQL). It runs as the one-shot `tls-init` subcommand before the
// other services start.
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
// pg_hba.conf into dir. Existing material is kept while it is complete, still
// matches opts, and is not close to expiry; otherwise everything is reissued.
// The CA private key is never written to disk.
func Ensure(dir string, opts Options) (issued bool, err error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if len(opts.DBHosts) == 0 || len(opts.BackendHosts) == 0 {
		return false, errors.New("db and backend hosts are required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	if current(dir, opts) {
		return false, writeFile(filepath.Join(dir, PGHBAFile), []byte(PGHBA), 0o644, -1, -1)
	}
	now := opts.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return false, err
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "Roomies internal CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return false, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return false, err
	}
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
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	if err := writeFile(filepath.Join(dir, CAFile), caPEM, 0o644, -1, -1); err != nil {
		return false, err
	}
	return true, writeFile(filepath.Join(dir, PGHBAFile), []byte(PGHBA), 0o644, -1, -1)
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

// current reports whether every file exists, both leaves chain to the CA for
// their configured hosts, and nothing expires within renewThreshold.
func current(dir string, opts Options) bool {
	caCert, err := readCert(filepath.Join(dir, CAFile))
	if err != nil || opts.Now().Add(renewThreshold).After(caCert.NotAfter) {
		return false
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	check := func(certFile, keyFile string, hosts []string) bool {
		if _, err := os.Stat(filepath.Join(dir, keyFile)); err != nil {
			return false
		}
		cert, err := readCert(filepath.Join(dir, certFile))
		if err != nil || opts.Now().Add(renewThreshold).After(cert.NotAfter) {
			return false
		}
		for _, host := range hosts {
			if _, err := cert.Verify(x509.VerifyOptions{DNSName: host, Roots: roots, CurrentTime: opts.Now()}); err != nil {
				return false
			}
		}
		return true
	}
	return check(DBCertFile, DBKeyFile, opts.DBHosts) && check(BackendCert, BackendKey, opts.BackendHosts)
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

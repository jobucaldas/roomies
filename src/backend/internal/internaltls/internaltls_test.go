package internaltls

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureIssuesVerifiableCertificatesOnce(t *testing.T) {
	dir := t.TempDir()
	opts := Options{DBHosts: []string{"db"}, BackendHosts: []string{"backend", "127.0.0.1"}, DBGroup: -1, BackendUser: -1}
	issued, err := Ensure(dir, opts)
	if err != nil || !issued {
		t.Fatalf("first run: issued=%v err=%v", issued, err)
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, CAFile))
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(caPEM)
	for certFile, keyFile := range map[string]string{DBCertFile: DBKeyFile, BackendCert: BackendKey} {
		pair, err := tls.LoadX509KeyPair(filepath.Join(dir, certFile), filepath.Join(dir, keyFile))
		if err != nil {
			t.Fatal(err)
		}
		leaf, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			t.Fatal(err)
		}
		hosts := opts.BackendHosts
		if certFile == DBCertFile {
			hosts = opts.DBHosts
		}
		for _, host := range hosts {
			if _, err := leaf.Verify(x509.VerifyOptions{DNSName: host, Roots: roots}); err != nil {
				t.Fatalf("%s does not verify for %s: %v", certFile, host, err)
			}
		}
		info, err := os.Stat(filepath.Join(dir, keyFile))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o007 != 0 {
			t.Fatalf("%s is world-accessible: %v", keyFile, info.Mode())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, PGHBAFile)); err != nil {
		t.Fatal(err)
	}

	issued, err = Ensure(dir, opts)
	if err != nil || issued {
		t.Fatalf("second run should keep existing material: issued=%v err=%v", issued, err)
	}
	opts.DBHosts = []string{"postgres"}
	if issued, err = Ensure(dir, opts); err != nil || !issued {
		t.Fatalf("changed hosts should reissue: issued=%v err=%v", issued, err)
	}
	opts.Now = func() time.Time { return time.Now().Add(validity) }
	if issued, err = Ensure(dir, opts); err != nil || !issued {
		t.Fatalf("expiring material should be reissued: issued=%v err=%v", issued, err)
	}
}

package securelink

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math"
	"math/big"
	// "math/rand"
	"net"
	"time"
)

// Those constants define used algorithm inside the package
const (
	SignatureAlgorithm = x509.ECDSAWithSHA384
	PublicKeyAlgorithm = x509.ECDSA
)

// Curve defines the elliptic curve used inside the package
var (
	Curve = elliptic.P384()
)

// GetCertTemplate returns the base template for certification
func GetCertTemplate(isCA bool, names []string, ips []net.IP, expireIn time.Duration) *x509.Certificate {
	serial, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	names = append(names, serial.String())

	return &x509.Certificate{
		SignatureAlgorithm: SignatureAlgorithm,

		SerialNumber: serial,
		Subject:      getSubject(),

		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(expireIn), // Validity bounds.
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageContentCommitment | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyAgreement | x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageEncipherOnly | x509.KeyUsageDecipherOnly,

		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}, // Sequence of extended key usages.

		// BasicConstraintsValid indicates whether IsCA, MaxPathLen,
		// and MaxPathLenZero are valid.
		BasicConstraintsValid: true,
		IsCA: isCA,

		// MaxPathLen and MaxPathLenZero indicate the presence and
		// value of the BasicConstraints' "pathLenConstraint".
		//
		// When parsing a certificate, a positive non-zero MaxPathLen
		// means that the field was specified, -1 means it was unset,
		// and MaxPathLenZero being true mean that the field was
		// explicitly set to zero. The case of MaxPathLen==0 with MaxPathLenZero==false
		// should be treated equivalent to -1 (unset).
		//
		// When generating a certificate, an unset pathLenConstraint
		// can be requested with either MaxPathLen == -1 or using the
		// zero value for both MaxPathLen and MaxPathLenZero.
		MaxPathLen: 0,
		// MaxPathLenZero indicates that BasicConstraintsValid==true
		// and MaxPathLen==0 should be interpreted as an actual
		// maximum path length of zero. Otherwise, that combination is
		// interpreted as MaxPathLen not being set.
		MaxPathLenZero: true, // Go 1.4

		DNSNames:    names,
		IPAddresses: ips, // Go 1.1
	}
}

func getSubject() pkix.Name {
	return pkix.Name{
		// Country:            []string{},
		// Organization:       []string{},
		// OrganizationalUnit: []string{},
		// Locality:           []string{},
		// Province:           []string{},
		// StreetAddress:      []string{},
		// PostalCode:         []string{},
		// SerialNumber:       "",
		CommonName: "go-db",
	}
}

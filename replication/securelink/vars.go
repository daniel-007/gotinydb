package securelink

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math"
	"math/big"
	"net"
	"time"
)

// Defines the supported key type
const (
	KeyTypeRSA KeyType = "RSA"
	KeyTypeEc  KeyType = "Elliptic Curve"
)

// Defines the supported key length
const (
	KeyLengthRsa2048 KeyLength = "RSA 2048"
	KeyLengthRsa3072 KeyLength = "RSA 3072"
	KeyLengthRsa4096 KeyLength = "RSA 4096"
	KeyLengthRsa8192 KeyLength = "RSA 8192"

	KeyLengthEc256 KeyLength = "EC 256"
	KeyLengthEc384 KeyLength = "EC 384"
	KeyLengthEc521 KeyLength = "EC 521"
)

// Those variables defines the most common package errors
var (
	ErrKeyConfigNotCompatible = fmt.Errorf("the key type and key size are not compatible")
)

// GetCertTemplate returns the base template for certification
func GetCertTemplate(names []string, ips []net.IP) *x509.Certificate {
	serial, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))

	if len(names) == 0 || names == nil {
		names = []string{}
	}
	names = append(names, serial.String())

	return &x509.Certificate{
		SignatureAlgorithm: x509.UnknownSignatureAlgorithm,

		SerialNumber: serial,
		Subject:      getSubject(),

		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365), // Validity bounds.
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageContentCommitment | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyAgreement | x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageEncipherOnly | x509.KeyUsageDecipherOnly,

		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}, // Sequence of extended key usages.

		// BasicConstraintsValid indicates whether IsCA, MaxPathLen,
		// and MaxPathLenZero are valid.
		BasicConstraintsValid: true,
		IsCA: false,

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
		DNSNames:       names,
		IPAddresses:    ips, // Go 1.1
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
		CommonName: "secure-link",
	}
}

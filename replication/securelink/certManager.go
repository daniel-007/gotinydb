package securelink

import (
	"crypto/elliptic"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
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

func getCertTemplate(isCA bool, names []string, ips []net.IP, expireIn time.Duration) *x509.Certificate {
	return &x509.Certificate{
		SignatureAlgorithm: SignatureAlgorithm,

		SerialNumber: big.NewInt(0),
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

		// SubjectKeyId   []byte
		// AuthorityKeyId []byte

		// // RFC 5280, 4.2.2.1 (Authority Information Access)
		// OCSPServer            []string // Go 1.2
		// IssuingCertificateURL []string // Go 1.2

		// // Subject Alternate Name values. (Note that these values may not be valid
		// // if invalid values were contained within a parsed certificate. For
		// // example, an element of DNSNames may not be a valid DNS domain name.)
		DNSNames: names,
		// EmailAddresses: []string{},
		IPAddresses: ips, // Go 1.1
		// URIs:           []*url.URL{}, // Go 1.10

		// // // Name constraints
		// PermittedDNSDomainsCritical: false, // if true then the name constraints are marked critical.
		// PermittedDNSDomains:         []string{},
		// ExcludedDNSDomains:          []string{},     // Go 1.9
		// PermittedIPRanges:           []*net.IPNet{}, // Go 1.10
		// ExcludedIPRanges:            []*net.IPNet{}, // Go 1.10
		// PermittedEmailAddresses:     []string{},     // Go 1.10
		// ExcludedEmailAddresses:      []string{},     // Go 1.10
		// PermittedURIDomains:         []string{},     // Go 1.10
		// ExcludedURIDomains:          []string{},     // Go 1.10

		// // CRL Distribution Points
		// CRLDistributionPoints: []string{}, // Go 1.2

		// PolicyIdentifiers: []asn1.ObjectIdentifier{},
	}
}

// func getReqTemplate(names []string, ips []net.IP) *x509.CertificateRequest {
// 	return &x509.CertificateRequest{
// 		SignatureAlgorithm: SignatureAlgorithm,

// 		PublicKeyAlgorithm: PublicKeyAlgorithm,

// 		Subject: getSubject(),

// 		// // Attributes is the dried husk of a bug and shouldn't be used.
// 		// Attributes []pkix.AttributeTypeAndValueSET

// 		// // ExtraExtensions contains extensions to be copied, raw, into any
// 		// // marshaled CSR. Values override any extensions that would otherwise
// 		// // be produced based on the other fields but are overridden by any
// 		// // extensions specified in Attributes.
// 		// //
// 		// // The ExtraExtensions field is not populated when parsing CSRs, see
// 		// // Extensions.
// 		// ExtraExtensions []pkix.Extension

// 		// Subject Alternate Name values.
// 		DNSNames: names,
// 		// EmailAddresses: []string{},
// 		IPAddresses: ips,
// 		// URIs:           []*url.URL{}, // Go 1.10
// 	}
// }

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

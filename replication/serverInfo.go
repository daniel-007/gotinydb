package replication

// func (n *node) buildNewConnectionRequest() *newConnectionRequest {
// 	certSignature := make([]byte, len(n.Certificate.Cert.Signature))
// 	copy(certSignature, n.Certificate.Cert.Signature)

// 	return &newConnectionRequest{
// 		ID:              uuid.NewV4().String(),
// 		IssuerID:        n.ID,
// 		IssuerPort:      n.Port,
// 		IssuerAddresses: n.GetAddresses(),
// 		CACertSignature: certSignature,
// 	}
// }

// func (n *node) buildServerInfoForToken() (requestID string, reqAsJSON []byte) {
// 	req := n.buildNewConnectionRequest()
// 	reqAsJSON, _ = json.Marshal(req)
// 	return req.ID, reqAsJSON
// }

package replication_test

import (
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication"
	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestNodes(t *testing.T) {
	ca, _ := securelink.NewCA(time.Hour, "master")

	masterNode, err := replication.NewMasterNode(ca, ":1323")
	if err != nil {
		t.Fatal(err)
	}
	defer masterNode.Close()

	go masterNode.Start()

	token, err := masterNode.GetToken()
	if err != nil {
		t.Fatal(err)
	}

	var node1 replication.Node
	node1, err = replication.Connect(token, ":1324")
	if err != nil {
		t.Fatal(err)
	}
	defer node1.Close()

	_, err = replication.Connect(token, ":1325")
	if err == nil {
		t.Fatal("the token must be expired")
	}

	token, err = masterNode.GetToken()
	if err != nil {
		t.Fatal(err)
	}

	var node2 replication.Node
	node2, err = replication.Connect(token, ":1325")
	if err != nil {
		t.Fatal(err)
	}
	defer node2.Close()

	// data := url.Values{}
	// data.Set("token", token)

	// path := fmt.Sprintf("https://%s%s/%s%s", masterNode.GetAddresses()[0], masterNode.GetPort(), replication.APIVersion, replication.GetCertificatePATH)
	// fmt.Println("path", path)

	// connector := securelink.NewConnector("master", server1Cert)
	// connector.PostForm(
	// 	path,
	// 	data,
	// )

	// insecureClient := &http.Client{
	// 	Transport: &http.Transport{
	// 		TLSClientConfig: &tls.Config{
	// 			ServerName: masterNode.GetID(),
	// 			RootCAs:    ca.GetCertPool(),
	// 		},
	// 	},
	// }
	// resp, err := insecureClient.Get(path)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println("Status", resp.Status)
	// resp, err = insecureClient.Get(path)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println("Status", resp.Status)
}

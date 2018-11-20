package replication_test

// func TestRaftNode(t *testing.T) {
// 	ca, _ := securelink.NewCA(time.Hour, "master")

// 	masterNode, err := replication.NewMasterNode(ca, ":1323")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	go masterNode.GetServer().Start()

// 	token, err := masterNode.GetToken()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	var node1 replication.Node
// 	node1, err = replication.Connect(token, ":1324")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	go node1.GetServer().Start()

// 	token, err = masterNode.GetToken()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	var node2 replication.Node
// 	node2, err = replication.Connect(token, ":1325")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	go node2.GetServer().Start()

// 	time.Sleep(time.Second * 5)
// }

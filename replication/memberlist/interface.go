package memberlist

import (
	"net"

	"github.com/hashicorp/memberlist"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

type (
	Transport struct {
		server      *securelink.Server
		handlerChan chan net.Conn
		packetChan  chan *memberlist.Packet
	}

	// customConn struct {
	// 	*securelink.TransportConn
	// 	packetChan chan *memberlist.Packet
	// }
)

func NewMemberlistTransport(s *securelink.Server) *Transport {
	return &Transport{
		server:      s,
		handlerChan: make(chan net.Conn, 0),
		packetChan:  make(chan *memberlist.Packet, 0),
	}
}

// // FinalAdvertiseAddr is given the user's configured values (which
// // might be empty) and returns the desired IP and port to advertise to
// // the rest of the cluster.
// func (t *Transport) FinalAdvertiseAddr(ip string, port int) (net.IP, int, error) {
// 	return t.server.AddrStruct.IP(), int(t.server.AddrStruct.Port), nil
// }

// // WriteTo is a packet-oriented interface that fires off the given
// // payload to the given address in a connectionless fashion. This should
// // return a time stamp that's as close as possible to when the packet
// // was transmitted to help make accurate RTT measurements during probes.
// //
// // This is similar to net.PacketConn, though we didn't want to expose
// // that full set of required methods to keep assumptions about the
// // underlying plumbing to a minimum. We also treat the address here as a
// // string, similar to Dial, so it's network neutral, so this usually is
// // in the form of "host:port".
// func (t *Transport) WriteTo(b []byte, addr string) (time.Time, error) {
// 	conn, err := t.DialTimeout(addr, time.Second*30)
// 	if err != nil {
// 		return time.Now(), err
// 	}

// 	_, err = conn.Write(b)
// 	return time.Now(), err
// }

// // PacketCh returns a channel that can be read to receive incoming
// // packets from other peers. How this is set up for listening is left as
// // an exercise for the concrete transport implementations.
// func (t *Transport) PacketCh() <-chan *memberlist.Packet {
// 	fmt.Println("PacketCh")
// 	return t.packetChan
// }

// // DialTimeout is used to create a connection that allows us to perform
// // two-way communication with a peer. This is generally more expensive
// // than packet connections so is used for more infrequent operations
// // such as anti-entropy or fallback probes if the packet-oriented probe
// // failed.
// func (t *Transport) DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
// 	conn, err := t.server.Dial(addr, "memberlist", timeout)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// tlsConn, ok := conn.(*tls.Conn)
// 	// if !ok {
// 	// 	return nil, fmt.Errorf("this is not a TLS connection")
// 	// }

// 	// err = tlsConn.Handshake()
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	return conn, err
// }

// // StreamCh returns a channel that can be read to handle incoming stream
// // connections from other peers. How this is set up for listening is
// // left as an exercise for the concrete transport implementations.
// func (t *Transport) StreamCh() <-chan net.Conn {
// 	fmt.Println("StreamCh")
// 	return t.handlerChan
// }

// // Shutdown is called when memberlist is shutting down; this gives the
// // transport a chance to clean up any listeners.
// func (t *Transport) Shutdown() error {
// 	return nil
// }

// func (t *Transport) Handle(conn *securelink.TransportConn) error {
// 	t0 := time.Now()
// 	fmt.Println("handle memberlist")

// 	buff := make([]byte, 4096)
// 	n, err := conn.Read(buff)
// 	if err != nil {
// 		return err
// 	}

// 	defer conn.Close()

// 	buff = buff[:n]
// 	packet := &memberlist.Packet{
// 		Buf:       buff,
// 		From:      conn.RemoteAddr(),
// 		Timestamp: t0,
// 	}

// 	t.packetChan <- packet

// 	// cc := &customConn{
// 	// 	TransportConn: conn,
// 	// 	packetChan:    t.packetChan,
// 	// }
// 	t.handlerChan <- conn

// 	fmt.Println("t.packetChan", len(t.packetChan))
// 	fmt.Println("t.handlerChan", len(t.handlerChan))

// 	conn.Wait()

// 	return conn.Error()
// }

// func (cc *customConn) Read(b []byte) (int, error) {
// 	fmt.Println("11 read")

// 	t0 := time.Now()

// 	n, err := cc.TransportConn.Read(b)
// 	if err != nil {
// 		return 0, err
// 	}

// 	cp := make([]byte, n)
// 	copy(cp, b)

// 	fmt.Println("read", string(cp))

// 	packet := &memberlist.Packet{
// 		Buf:       cp,
// 		From:      cc.RemoteAddr(),
// 		Timestamp: t0,
// 	}

// 	cc.packetChan <- packet

// 	return n, err
// }

// func (cc *customConn) Write(b []byte) (int, error) {
// 	fmt.Println("read", string(b))
// 	return cc.TransportConn.Write(b)
// }

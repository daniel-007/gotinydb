package common

import (
	"fmt"
	"net"
)

type (
	Addr struct {
		MainAddr string
		Port     uint16
		Addrs    []string
	}
)

func NewAddr(port uint16) (*Addr, error) {
	addrs, err := GetAddresses()
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("no address found")
	}

	return &Addr{
		MainAddr: addrs[0],
		Port:     port,
		Addrs:    addrs,
	}, nil
}

func (a *Addr) SwitchMain(i int) string {
	if i > len(a.Addrs)-1 {
		return ""
	}
	a.MainAddr = a.Addrs[i]
	return a.String()
}

func (a *Addr) String() string {
	return fmt.Sprintf("%s:%d", a.MainAddr, a.Port)
}

func (a *Addr) Network() string {
	return "tcp"
}

func (a *Addr) ForListenerBroadcast() string {
	return fmt.Sprintf(":%d", a.Port)
}

func (a *Addr) IP() net.IP {
	return net.ParseIP(a.MainAddr)
}

func GetAddresses() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ret := []string{}

	for _, nic := range interfaces {
		var addrs []net.Addr
		addrs, err = nic.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ipAsString := addr.String()
			ip, _, err := net.ParseCIDR(ipAsString)
			if err != nil {
				continue
			}

			ipAsString = ip.String()
			ip2 := net.ParseIP(ipAsString)
			if to4 := ip2.To4(); to4 == nil {
				ipAsString = "[" + ipAsString + "]"
			}

			// If ip accessible from outside
			if ip.IsGlobalUnicast() {
				ret = append(ret, ipAsString)
			}
		}
	}

	return ret, nil
}

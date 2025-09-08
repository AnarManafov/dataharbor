package util

import (
	"errors"
	"net"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	snowNode *snowflake.Node
	once     sync.Once
)

// InitSnowflake initializes the snowflake node for ID generation
// This should be called once during application startup
func InitSnowflake() error {
	var initErr error
	once.Do(func() {
		localIp, err := getClientIp()
		if err != nil {
			initErr = err
			return
		}
		nodeId, err := Ipv4ToLong(localIp)
		if err != nil {
			initErr = err
			return
		}

		id := nodeId % 1024
		snowNode, err = snowflake.NewNode(int64(id))
		if err != nil {
			initErr = err
			return
		}
	})
	return initErr
}

func Ipv4ToLong(ip string) (uint, error) {
	p := net.ParseIP(ip).To4()
	if p == nil {
		return 0, errors.New("invalid ipv4 format")
	}

	return uint(p[0])<<24 | uint(p[1])<<16 | uint(p[2])<<8 | uint(p[3]), nil
}

func getClientIp() (string, error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}

		}
	}

	return "", errors.New("cannot find the client ip address")

}

func NextUid() string {
	// Ensure snowflake is initialized (in case it wasn't explicitly called)
	if snowNode == nil {
		if err := InitSnowflake(); err != nil {
			// If initialization fails, we cannot generate IDs
			// This is a critical error that should be handled by the caller
			panic("failed to initialize snowflake: " + err.Error())
		}
	}
	// Generate a snowflake ID.
	return snowNode.Generate().String()
}

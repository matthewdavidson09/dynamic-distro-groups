package ldapclient

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/joho/godotenv"
	"github.com/matthewdavidson09/dynamic-distro-groups/tools"
)

type LDAPClient struct {
	Conn   *ldap.Conn
	BaseDN string
}

// Connect resolves LDAP hostname to an IP and returns a bound LDAPClient.
func Connect() (*LDAPClient, error) {
	if err := godotenv.Load(".env"); err != nil {
		return nil, fmt.Errorf("error loading .env: %w", err)
	}

	server := strings.TrimSpace(os.Getenv("LDAP_SERVER"))
	port := os.Getenv("LDAP_PORT")
	if port == "" {
		port = "389"
	}

	// Resolve DNS
	addrs, err := net.LookupHost(server)
	if err != nil || len(addrs) == 0 {
		return nil, fmt.Errorf("DNS lookup failed for %s: %v", server, err)
	}
	ip := addrs[0]

	tools.Log.WithFields(map[string]interface{}{
		"host": server,
		"ip":   ip,
		"port": port,
	}).Debug("Resolved LDAP server IP")

	return ConnectWithIP(ip, port)
}

// ConnectWithIP connects to a specific LDAP IP and returns a bound client.
func ConnectWithIP(ip, port string) (*LDAPClient, error) {
	user := strings.TrimSpace(os.Getenv("LDAP_USER"))
	pass := strings.TrimSpace(os.Getenv("LDAP_PASSWORD"))
	baseDN := strings.TrimSpace(os.Getenv("BASE_DN"))

	url := fmt.Sprintf("ldap://%s:%s", ip, port)
	tools.Log.WithField("url", url).Debug("Connecting to resolved LDAP IP")

	conn, err := ldap.DialURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}

	if err := conn.Bind(user, pass); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind: %w", err)
	}

	tools.Log.Debug("Successfully bound to LDAP")

	return &LDAPClient{
		Conn:   conn,
		BaseDN: baseDN,
	}, nil
}

// Close cleans up the connection
func (c *LDAPClient) Close() {
	if c.Conn != nil {
		c.Conn.Close()
		tools.Log.Debug("Closed LDAP connection")
	}
}

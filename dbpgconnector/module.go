package dbpgconnector

import (
	"context"
	"database/sql"
	"time"

	"github.com/dm1trypon/db-mdl/dbpgtools"
)

// DBPGConnector - main struct of the DB's module
type DBPGConnector struct {
	lc            string  // logging category
	conn          *sql.DB // data of DBPG connection
	ctx           context.Context
	config        Config // DBPG settings
	sslModes      map[uint8]string
	dbPgToolsList map[sql.TxOptions]*dbpgtools.DBPGTools
}

// Config - DBPG settings
type Config struct {
	Username string // username
	Password string // password
	Host     string // host
	Port     uint16 // port
	DbName   string // database name
	// Valid values for sslmode are:
	//
	// 0 - disable - No SSL
	//
	// 1 - require - Always SSL (skip verification)
	//
	// 2 - verify-ca - Always SSL (verify that the certificate presented by the
	// server was signed by a trusted CA)
	//
	// 3 - verify-full - Always SSL (verify that the certification presented by
	// the server was signed by a trusted CA and the server host name
	// matches the one in the certificate)
	SSLMode              uint8
	ConnectTimeout       uint16        // timeout connection for Postgres
	PingInterval         time.Duration // interval for sending Ping
	ReconnectionInterval time.Duration // interval for reconnect
	Certs                Certs         // data of Certificates, needed for TLS connection
}

// Certs - data of Certificates, needed for TLS connection
type Certs struct {
	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name. If InsecureSkipVerify is true, crypto/tls
	// accepts any certificate presented by the server and any host name in that
	// certificate. In this mode, TLS is susceptible to machine-in-the-middle
	// attacks unless custom verification is used. This should be used only for
	// testing or in combination with VerifyConnection or VerifyPeerCertificate.
	InsecureSkipVerify bool
	Srcs               Srcs  // sourses data of certs
	Paths              Paths // data of certs paths
}

// Srcs - sourses data of certs
type Srcs struct {
	CA      []byte // root certificate
	SrvCert []byte // public certificate
	SrvKey  []byte // public key
}

// Paths - data of certs paths
type Paths struct {
	CA      string // root certificate
	SrvCert string // public certificate
	SrvKey  string // public key
}

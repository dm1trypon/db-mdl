package db

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	logger "github.com/dm1trypon/easy-logger"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Intervals
const (
	// DefaultReconnectInterval - default reconnecting interval
	DefaultReconnectInterval = 1 // sec
	// DefaultPingInterval - default ping interval
	DefaultPingInterval = 1 // sec
)

// SQL's settings
const (
	// DefaultDriver - default driver for SQL server
	DefaultDriver = "mysql"
)

// Certs
const (
	// DefaultPathCACrt - default path to CA certificate
	DefaultPathCACrt = "db-ca-cert.pem"
	// DefaultPathCrt - default path to client's certificate
	DefaultPathCrt = "db-client-cert.pem"
	// DefaultPathKey - default path to key
	DefaultPathKey = "db-client-key.pem"
	// DefaultSSLFolder - default SSL folder
	DefaultSSLFolder = "./ssl/"
)

// SQLResult - overriding a query result data variable
type SQLResult *sqlx.Rows

// Create - DataBase's constructor
func (d *DataBase) Create() *DataBase {
	d = &DataBase{
		lc:                "DATABASE",
		mutex:             &sync.Mutex{},
		conn:              nil,
		isBusy:            false,
		reconnectInterval: DefaultReconnectInterval,
		pingInterval:      DefaultPingInterval,
		err:               make(chan error),
		connected:         make(chan bool),
		disconnected:      make(chan bool),
		username:          "",
		password:          "",
		host:              "",
		port:              0,
		dbName:            "",
		tls:               false,
		driver:            DefaultDriver,
		params:            "",
		caCrt:             "",
		crt:               "",
		key:               "",
		pathCACrt:         DefaultPathCACrt,
		pathCrt:           DefaultPathCrt,
		pathKey:           DefaultPathKey,
	}

	return d
}

// Run - starts the module.
// Returns <bool>. True - is ok.
func (d *DataBase) Run() bool {
	if d.tls {
		logger.Info(d.lc, "TLS enabled")

		if len(d.params) > 0 {
			d.params += "&tls=custom"
		} else {
			d.params += "tls=custom"
		}

		if len(d.caCrt) > 0 && len(d.crt) > 0 && len(d.key) > 0 {
			if !d.makeCerts() {
				return false
			}
		}

		if !d.configuring() {
			return false
		}
	}

	if len(d.driver) < 1 {
		d.driver = DefaultDriver
	}

	go d.connector()

	if d.pingInterval < 1 {
		return true
	}

	go d.inspector()

	return true
}

/*
SetPingInterval - sets the interval for checking the connection to the database.
	- Args:
		- pingInterval <int> - interval.
*/
func (d *DataBase) SetPingInterval(pingInterval int) {
	d.pingInterval = pingInterval
}

/*
SetHost - set the database host.
	- Args:
		- host <string> - address.
*/
func (d *DataBase) SetHost(host string) {
	d.host = host
}

/*
SetPort - set the database port.
	- Args:
		- port <int> - port.
*/
func (d *DataBase) SetPort(port int) {
	d.port = port
}

/*
SetUsername - set the username for connecting to the database.
	- Args:
		- username <string> - username or login.
*/
func (d *DataBase) SetUsername(username string) {
	d.username = username
}

/*
SetPassword - set the password for connecting to the database.
	- Args:
		- password <string> - password.
*/
func (d *DataBase) SetPassword(password string) {
	d.password = password
}

/*
SetDatabaseName - set the DB's name for connecting to the database.
	- Args:
		- dbName <string> - DB's name.
*/
func (d *DataBase) SetDatabaseName(dbName string) {
	d.dbName = dbName
}

/*
SetTLS - using TLS.
	- Args:
		- tls <bool> - using tls.
*/
func (d *DataBase) SetTLS(tls bool) {
	d.tls = tls
}

/*
SetDriver - configures which driver will be used. Default is 'mysql'.
	- Args:
		- driver <string> - SQL's driver.
*/
func (d *DataBase) SetDriver(driver string) {
	d.driver = driver
}

/*
SetParams - configures additional parameters for connecting to the DB.
If TLS is used then 'tls=custom' is added.
	- Args:
		- params <string> - parameters for DB.
*/
func (d *DataBase) SetParams(params string) {
	d.params = params
}

/*
SetCertsSrc - set the sourse data of the certificates.
	- Args:
		- caCrt <string> - CA cert.
		- crt <string> - root cert.
		- key <string> - key.
*/
func (d *DataBase) SetCertsSrc(caCrt, crt, key string) {
	d.caCrt = caCrt
	d.crt = crt
	d.key = key
}

/*
SetCertsPath - set paths of the certificates.
	- Args:
		- pathCACrt <string> - path to CA cert.
		- pathCrt <string> - path to root cert.
		- pathKey <string> - path to key.
*/
func (d *DataBase) SetCertsPath(pathCACrt, pathCrt, pathKey string) {
	d.pathCACrt = pathCACrt
	d.pathCrt = pathCrt
	d.pathKey = pathKey
}

// GetErrorEvent - error events.
// Returns <chan bool>.
func (d *DataBase) GetErrorEvent() chan error {
	return d.err
}

// GetConnectedEvent - connect events.
// Returns <chan bool>.
func (d *DataBase) GetConnectedEvent() chan bool {
	return d.connected
}

// GetDisconnectedEvent - disconnect events.
// Returns <chan bool>.
func (d *DataBase) GetDisconnectedEvent() chan bool {
	return d.disconnected
}

// inspector - DB connection check
func (d *DataBase) inspector() {
	for {
		if d.conn == nil {
			time.Sleep(time.Duration(d.pingInterval) * time.Second)
			continue
		}

		if err := d.conn.Ping(); err != nil {
			logger.Error(d.lc, fmt.Sprint("Ping server error: ", err.Error()))

			d.mutex.Lock()
			if err := d.conn.Close(); err != nil {
				logger.Error(d.lc, fmt.Sprint("Connection close error: ", err.Error()))
			}
			d.mutex.Unlock()
			logger.Debug(d.lc, fmt.Sprint("PING"))
			d.disconnected <- true
		}

		time.Sleep(time.Duration(d.pingInterval) * time.Second)
	}
}

// connector - DB connector
func (d *DataBase) connector() {
	for {
		go d.connect()
		<-d.disconnected
		time.Sleep(time.Duration(d.reconnectInterval) * time.Second)
	}
}

// connect - establishes a connection to the database
func (d *DataBase) connect() {
	// checking if the connection is already in progress
	if d.isBusy {
		return
	}

	var err error

	path := fmt.Sprint(d.username, ":", d.password, "@tcp(", d.host, ")/", d.dbName, "?", d.params)

	logger.Info(d.lc, fmt.Sprint("Connecting to DB: ", d.host, ":", d.port, "/", d.dbName))

	d.isBusy = true
	d.conn, err = sqlx.Connect(d.driver, path)

	if err != nil {
		logger.Error(d.lc, fmt.Sprint("Error in connection to DB: ", err.Error()))
		d.isBusy = false
		d.disconnected <- true
		return
	}

	logger.Info(d.lc, "Successful connection to DB")

	d.isBusy = false
	d.connected <- true
}

// makeCerts - creates certificates and key by env value.
// Returns <bool>. True - is ok.
func (d *DataBase) makeCerts() bool {
	if err := os.MkdirAll(DefaultSSLFolder, os.ModePerm); err != nil {
		logger.Error(d.lc, fmt.Sprint("Failed to create ssl directory: ", err.Error()))
		return false
	}

	var certs = map[string]string{
		fmt.Sprint(DefaultSSLFolder, DefaultPathCACrt): d.caCrt,
		fmt.Sprint(DefaultSSLFolder, DefaultPathCrt):   d.crt,
		fmt.Sprint(DefaultSSLFolder, DefaultPathKey):   d.key,
	}

	for path, data := range certs {
		file, err := os.Create(path)
		if err != nil {
			logger.Error(d.lc, fmt.Sprint("Unable to create certificate ", path, ": ", err.Error()))
			return false
		}

		file.WriteString(data)
		file.Close()

		logger.Info(d.lc, fmt.Sprint("Certificate ", path, " created"))
	}

	d.pathCACrt = fmt.Sprint(DefaultSSLFolder, DefaultPathCACrt)
	d.pathCrt = fmt.Sprint(DefaultSSLFolder, DefaultPathCrt)
	d.pathKey = fmt.Sprint(DefaultSSLFolder, DefaultPathKey)

	return true
}

// configuring - configures a TLS connection using certificates and keys located by path.
// Returns <bool>. True - is ok.
func (d *DataBase) configuring() bool {
	rootCertPool := x509.NewCertPool()

	pem, err := ioutil.ReadFile(d.pathCACrt)
	if err != nil {
		logger.Error(d.lc, fmt.Sprint("Error reading certs: ", err.Error()))
		return false
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		logger.Error(d.lc, "Failed to append PEM")
		return false
	}

	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(d.pathCrt, d.pathKey)
	if err != nil {
		logger.Error(d.lc, fmt.Sprint("LoadX509KeyPair reading error: ", err.Error()))
		return false
	}

	clientCert = append(clientCert, certs)

	mysql.RegisterTLSConfig("custom", &tls.Config{
		Rand:                        nil,
		Certificates:                clientCert,
		NameToCertificate:           map[string]*tls.Certificate{},
		RootCAs:                     rootCertPool,
		NextProtos:                  []string{},
		ServerName:                  "",
		ClientAuth:                  0,
		ClientCAs:                   &x509.CertPool{},
		InsecureSkipVerify:          true,
		CipherSuites:                []uint16{},
		PreferServerCipherSuites:    false,
		SessionTicketsDisabled:      false,
		SessionTicketKey:            [32]byte{},
		ClientSessionCache:          nil,
		MinVersion:                  0,
		MaxVersion:                  0,
		CurvePreferences:            []tls.CurveID{},
		DynamicRecordSizingDisabled: false,
		Renegotiation:               0,
		KeyLogWriter:                nil,
	})

	return true
}

/*
Queryx - execution of a query in the database.
Returns <SQLResult, error>.
	- Args:

		- query <string> - query.
*/
func (d *DataBase) Queryx(query string) (SQLResult, error) {
	logger.Debug(d.lc, fmt.Sprint("Query: ", query))

	if d.conn == nil {
		err := errors.New("Connection is not exist")
		logger.Error(d.lc, err.Error())
		return nil, err
	}

	d.mutex.Lock()
	rows, err := d.conn.Queryx(query)
	if err != nil {
		logger.Error(d.lc, fmt.Sprint("Error in running query: ", err.Error()))
		d.mutex.Unlock()
		return nil, err
	}
	d.mutex.Unlock()

	return rows, nil
}

package dbpgconnector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dm1trypon/db-mdl/dbpgtools"
	logger "github.com/dm1trypon/easy-logger"
	_ "github.com/lib/pq"
)

const MaxTxIsolationLvl = 7

func (d *DBPGConnector) Create() *DBPGConnector {
	d = &DBPGConnector{
		lc:   "DB_PG_CONNECTOR",
		conn: nil,
		ctx:  context.Background(),
		config: Config{
			Username:             "user",
			Password:             "password",
			Host:                 "localhost",
			Port:                 5432,
			DbName:               "db_test",
			ConnectTimeout:       10,
			PingInterval:         time.Second,
			ReconnectionInterval: time.Second,
			SSLMode:              0,
			Certs: Certs{
				InsecureSkipVerify: true,
				Srcs: Srcs{
					CA:      []byte{},
					SrvCert: []byte{},
					SrvKey:  []byte{},
				},
				Paths: Paths{
					CA:      "db-ca-cert.pem",
					SrvCert: "db-server-cert.pem",
					SrvKey:  "db-server-key.pem",
				},
			},
		},
		chConnected:    make(chan bool),
		chDisconnected: make(chan bool),
		sslModes: map[uint8]string{
			0: "disable",
			1: "require",
			2: "verify-ca",
			3: "verify-full",
		},
		dbPgToolsList: map[sql.TxOptions]*dbpgtools.DBPGTools{},
	}

	return d
}

func (d *DBPGConnector) GetChConnected() <-chan bool {
	return d.chConnected
}

func (d *DBPGConnector) GetChDisconnected() <-chan bool {
	return d.chDisconnected
}

func (d *DBPGConnector) SetDBPGToolsList(settings map[uint8]bool) {
	if len(d.dbPgToolsList) > 0 {
		d.dbPgToolsList = map[sql.TxOptions]*dbpgtools.DBPGTools{}
	}

	logger.DebugJ(d.lc, fmt.Sprint("SetDBPGToolsList"))

	for isolation, ro := range settings {
		if isolation > MaxTxIsolationLvl {
			logger.ErrorJ(d.lc, fmt.Sprint("Wrong transaction isolation level: ", isolation))
			continue
		}

		txOptions := sql.TxOptions{
			Isolation: sql.IsolationLevel(isolation),
			ReadOnly:  ro,
		}

		d.dbPgToolsList[txOptions] = new(dbpgtools.DBPGTools).Create(d.conn, d.ctx, txOptions)
	}
}

func (d *DBPGConnector) GetDBPGTools(isolationLvl uint8, ro bool) *dbpgtools.DBPGTools {
	for txOptions, dbPgTool := range d.dbPgToolsList {
		if txOptions.Isolation == sql.IsolationLevel(isolationLvl) &&
			txOptions.ReadOnly == ro {
			return dbPgTool
		}
	}

	return nil
}

func (d *DBPGConnector) setup() string {
	path := fmt.Sprint("user=", d.config.Username,
		" password=", d.config.Password,
		" host=", d.config.Host,
		" port=", d.config.Port,
		" dbname=", d.config.DbName,
		" connect_timeout=", d.config.ConnectTimeout,
	)

	if int(d.config.SSLMode) >= len(d.sslModes) {
		logger.WarningJ(d.lc, fmt.Sprint("SSLMode ", d.config.SSLMode, " is not supported, usind default 0"))
		d.config.SSLMode = 0
	}

	if d.config.SSLMode > 0 {
		if d.isSourceCerts() && !d.makeCerts() {
			return ""
		}

		path = fmt.Sprint(path, " sslmode=", d.sslModes[d.config.SSLMode],
			" sslcert=", d.config.Certs.Paths.SrvCert,
			" sslkey=", d.config.Certs.Paths.SrvKey,
			" sslrootcert=", d.config.Certs.Paths.CA)
	} else {
		path = fmt.Sprint(path, " sslmode=", d.sslModes[d.config.SSLMode])
	}

	return path
}

// SetConfig <*DBPGConnector> - sets DBPG settings
//
// Args:
// 	1. config <Config>
// - settings object for DBPG module
func (d *DBPGConnector) SetConfig(config Config) {
	d.config = config
}

// inspector - DB connection check
func (d *DBPGConnector) inspector() {
	for {
		logger.DebugJ(d.lc, fmt.Sprint("PING"))

		if d.conn == nil {
			break
		}

		if err := d.conn.PingContext(d.ctx); err != nil {
			logger.ErrorJ(d.lc, fmt.Sprint("Ping server error: ", err.Error()))

			if err := d.conn.Close(); err != nil {
				logger.ErrorJ(d.lc, fmt.Sprint("Connection close error: ", err.Error()))
			}

			time.Sleep(d.config.PingInterval)
			d.chDisconnected <- true
			d.Run()
			return
		}

		time.Sleep(d.config.PingInterval)
	}
}

func (d *DBPGConnector) Run() {
	for {
		if !d.RunOnce() {
			time.Sleep(d.config.ReconnectionInterval)
			continue
		}

		break
	}
}

func (d *DBPGConnector) RunOnce() bool {
	path := d.setup()

	if len(path) < 1 {
		return false
	}

	var err error

	logger.InfoJ(d.lc, fmt.Sprint("Connecting to DB: ", d.config.Host, ":", d.config.Port, "/", d.config.DbName))

	d.conn, err = sql.Open("postgres", path)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Error in connection to DB: ", err.Error()))
		return false
	}

	if err := d.conn.PingContext(d.ctx); err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Connection failed: ", err.Error()))
		d.conn.Close()
		return false
	}

	logger.InfoJ(d.lc, "Connected")

	go d.inspector()

	d.chConnected <- true

	return true
}

// makeCerts <*DBPGConnector> - creates certificates and key by env value
//
// Returns:
// 	1. <bool>
// - completion status
func (d *DBPGConnector) makeCerts() bool {
	if err := os.MkdirAll(filepath.Dir(d.config.Certs.Paths.CA), os.ModePerm); err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to create ssl directory: ", err.Error()))
		return false
	}

	var certs = map[string][]byte{
		d.config.Certs.Paths.CA:      d.config.Certs.Srcs.CA,
		d.config.Certs.Paths.SrvCert: d.config.Certs.Srcs.SrvCert,
		d.config.Certs.Paths.SrvKey:  d.config.Certs.Srcs.SrvKey,
	}

	for path, data := range certs {
		file, err := os.Create(path)
		if err != nil {
			logger.ErrorJ(d.lc, fmt.Sprint("Unable to create certificate ", path, ": ", err.Error()))
			return false
		}

		_, err = file.Write(data)
		if err != nil {
			logger.ErrorJ(d.lc, fmt.Sprint("Unable to write certificate ", path, ": ", err.Error()))
			return false
		}

		file.Close()

		logger.InfoJ(d.lc, fmt.Sprint("Certificate ", path, " created"))
	}

	return true
}

func (d *DBPGConnector) isSourceCerts() bool {
	return len(d.config.Certs.Srcs.CA) > 0 &&
		len(d.config.Certs.Srcs.SrvCert) > 0 &&
		len(d.config.Certs.Srcs.SrvKey) > 0
}

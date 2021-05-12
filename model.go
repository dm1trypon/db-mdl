package db

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

// DataBase - main struct of the DB's module
type DataBase struct {
	lc                string      // logging category
	mutex             *sync.Mutex // mutex for DB connection
	conn              *sqlx.DB    // data of DB connection
	isBusy            bool        // checking is busy connecting to DB
	reconnectInterval int         // reconnecting interval (seconds)
	pingInterval      int         // checking connection by interval (seconds)
	err               chan error  // error events
	connected         chan bool   // connected events
	disconnected      chan bool   // disconneted events
	username          string      // username
	password          string      // password
	host              string      // host
	port              int         // port
	dbName            string      // database's name
	tls               bool        // using TLS
	driver            string      // using SQL's driver. Default 'mysql'
	params            string      // additional connection parameters
	caCrt             string      // CA cert
	crt               string      // root cert
	key               string      // key
	pathCACrt         string      // path to CA cert
	pathCrt           string      // path to root cert
	pathKey           string      // path to key
}

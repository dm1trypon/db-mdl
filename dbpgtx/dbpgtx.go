package dbpgtx

import (
	"context"
	"database/sql"
	"fmt"

	logger "github.com/dm1trypon/easy-logger"
)

func (d *DBPGTx) Create(conn *sql.DB, ctx context.Context, txOptions sql.TxOptions) *DBPGTx {
	d = &DBPGTx{
		lc:        "DB_PG_TX",
		conn:      conn,
		sqlTx:     nil,
		ctx:       ctx,
		txOptions: txOptions,
	}

	return d
}

func (d *DBPGTx) Begin() bool {
	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return false
	}

	var err error
	d.sqlTx, err = d.conn.BeginTx(d.ctx, &d.txOptions)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed start transaction: ", err.Error()))
		return false
	}

	return true
}

func (d *DBPGTx) Commit() bool {
	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return false
	}

	if d.sqlTx == nil && !d.Begin() {
		return false
	}

	if err := d.sqlTx.Commit(); err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to commit transaction: ", err.Error()))
		return false
	}

	return true
}

func (d *DBPGTx) Rollback() bool {
	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return false
	}

	if d.sqlTx == nil && !d.Begin() {
		return false
	}

	if err := d.sqlTx.Rollback(); err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to rollback transcation: ", err.Error()))
		return false
	}

	return true
}
func (d *DBPGTx) Query(query string) (*sql.Rows, bool) {
	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return nil, false
	}

	if d.sqlTx == nil && !d.Begin() {
		return nil, false
	}

	sqlRows, err := d.sqlTx.QueryContext(d.ctx, query)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to execute query: ", err.Error()))
		return nil, false
	}

	return sqlRows, false
}

func (d *DBPGTx) Exec(query string) (sql.Result, bool) {
	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return nil, false
	}

	if d.sqlTx == nil && !d.Begin() {
		return nil, false
	}

	sqlResult, err := d.sqlTx.ExecContext(d.ctx, query)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to execute query: ", err.Error()))
		return nil, false
	}

	return sqlResult, true
}

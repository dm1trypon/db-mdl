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
func (d *DBPGTx) Query(query string) ([]map[string]interface{}, bool) {
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

	return d.makeResult(sqlRows)
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

func (d *DBPGTx) makeResult(sqlRows *sql.Rows) ([]map[string]interface{}, bool) {
	result := []map[string]interface{}{}

	columns, err := sqlRows.Columns()
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to get columns from SQL result: ", err.Error()))
		return nil, false
	}

	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)

	for sqlRows.Next() {
		for index := range columns {
			valuePtrs[index] = &values[index]
		}

		if err := sqlRows.Scan(valuePtrs...); err != nil {
			logger.WarningJ(d.lc, fmt.Sprint("Failed to scan values from SQL result's row: ", err.Error()))
			continue
		}

		rowRes := map[string]interface{}{}

		for i, col := range columns {
			val := values[i]

			b, ok := val.([]byte)
			var v interface{}
			if ok {
				v = string(b)
			} else {
				v = val
			}

			rowRes[col] = v
		}

		result = append(result, rowRes)
	}

	return result, true
}

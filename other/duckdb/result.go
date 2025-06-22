package duckdb

// result implements the driver.Result interface.
type result struct {
	lastInsertID int64
	rowsAffected int64
}

// LastInsertId returns the last inserted id, if applicable.
func (r result) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// RowsAffected returns the number of rows affected by the query.
func (r result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

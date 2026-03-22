package sqlstore

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

var errFakeDB = errors.New("fake db error")

func init() {
	sql.Register("fake-rows-err", &fakeRowsErrDriver{})
	sql.Register("fake-scan-err", &fakeScanErrDriver{})
	sql.Register("fake-commit-err", &fakeCommitErrDriver{})
	sql.Register("fake-rows-aff-err", &fakeRowsAffErrDriver{})
}

// openFakeDB opens a *sql.DB backed by the named fake driver.
func openFakeDB(t *testing.T, driverName string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open(%q): %v", driverName, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ── fake-rows-err ─────────────────────────────────────────────────────────────
// rows.Next() immediately returns errFakeDB so rows.Err() propagates it.

type (
	fakeRowsErrDriver struct{}
	fakeRowsErrConn   struct{}
	fakeRowsErrStmt   struct{}
	fakeErrRows       struct{}
)

func (d *fakeRowsErrDriver) Open(_ string) (driver.Conn, error) { return &fakeRowsErrConn{}, nil }
func (c *fakeRowsErrConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeRowsErrStmt{}, nil
}
func (c *fakeRowsErrConn) Close() error             { return nil }
func (c *fakeRowsErrConn) Begin() (driver.Tx, error) { return nil, errFakeDB }

func (s *fakeRowsErrStmt) Close() error                                  { return nil }
func (s *fakeRowsErrStmt) NumInput() int                                  { return -1 }
func (s *fakeRowsErrStmt) Exec(_ []driver.Value) (driver.Result, error)  { return nil, errFakeDB }
func (s *fakeRowsErrStmt) Query(_ []driver.Value) (driver.Rows, error)   { return &fakeErrRows{}, nil }

func (r *fakeErrRows) Columns() []string             { return nil }
func (r *fakeErrRows) Close() error                  { return nil }
func (r *fakeErrRows) Next(_ []driver.Value) error   { return errFakeDB }

// ── fake-scan-err ─────────────────────────────────────────────────────────────
// rows.Next() returns one row with a value that cannot be scanned into a string,
// triggering rows.Scan() to return an error.

type (
	fakeScanErrDriver struct{}
	fakeScanErrConn   struct{}
	fakeScanErrStmt   struct{}
	fakeScanErrRows   struct{ done bool }
)

func (d *fakeScanErrDriver) Open(_ string) (driver.Conn, error) { return &fakeScanErrConn{}, nil }
func (c *fakeScanErrConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeScanErrStmt{}, nil
}
func (c *fakeScanErrConn) Close() error             { return nil }
func (c *fakeScanErrConn) Begin() (driver.Tx, error) { return nil, errFakeDB }

func (s *fakeScanErrStmt) Close() error                                 { return nil }
func (s *fakeScanErrStmt) NumInput() int                                 { return -1 }
func (s *fakeScanErrStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, errFakeDB }
func (s *fakeScanErrStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeScanErrRows{}, nil
}

func (r *fakeScanErrRows) Columns() []string { return []string{"col"} }
func (r *fakeScanErrRows) Close() error      { return nil }
func (r *fakeScanErrRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	// A channel is not a valid driver.Value; database/sql's convertAssign will
	// fail to convert it into a *string, making rows.Scan return an error.
	dest[0] = make(chan struct{})
	return nil
}

// ── fake-commit-err ───────────────────────────────────────────────────────────
// ExecContext succeeds but tx.Commit() returns errFakeDB.

type (
	fakeCommitErrDriver struct{}
	fakeCommitErrConn   struct{}
	fakeCommitErrTx     struct{}
	fakeOKStmt          struct{}
	fakeOKResult        struct{}
)

func (d *fakeCommitErrDriver) Open(_ string) (driver.Conn, error) {
	return &fakeCommitErrConn{}, nil
}
func (c *fakeCommitErrConn) Prepare(_ string) (driver.Stmt, error) { return &fakeOKStmt{}, nil }
func (c *fakeCommitErrConn) Close() error                           { return nil }
func (c *fakeCommitErrConn) Begin() (driver.Tx, error)              { return &fakeCommitErrTx{}, nil }

func (tx *fakeCommitErrTx) Commit() error   { return errFakeDB }
func (tx *fakeCommitErrTx) Rollback() error { return nil }

func (s *fakeOKStmt) Close() error                                  { return nil }
func (s *fakeOKStmt) NumInput() int                                  { return -1 }
func (s *fakeOKStmt) Exec(_ []driver.Value) (driver.Result, error)  { return &fakeOKResult{}, nil }
func (s *fakeOKStmt) Query(_ []driver.Value) (driver.Rows, error)   { return nil, errFakeDB }

func (r *fakeOKResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeOKResult) RowsAffected() (int64, error) { return 1, nil }

// ── fake-rows-aff-err ─────────────────────────────────────────────────────────
// ExecContext returns a result where RowsAffected() returns errFakeDB.

type (
	fakeRowsAffErrDriver struct{}
	fakeRowsAffErrConn   struct{}
	fakeRowsAffErrStmt   struct{}
	fakeRowsAffErrResult struct{}
)

func (d *fakeRowsAffErrDriver) Open(_ string) (driver.Conn, error) {
	return &fakeRowsAffErrConn{}, nil
}
func (c *fakeRowsAffErrConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeRowsAffErrStmt{}, nil
}
func (c *fakeRowsAffErrConn) Close() error             { return nil }
func (c *fakeRowsAffErrConn) Begin() (driver.Tx, error) { return nil, errFakeDB }

func (s *fakeRowsAffErrStmt) Close() error                                  { return nil }
func (s *fakeRowsAffErrStmt) NumInput() int                                  { return -1 }
func (s *fakeRowsAffErrStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeRowsAffErrResult{}, nil
}
func (s *fakeRowsAffErrStmt) Query(_ []driver.Value) (driver.Rows, error) { return nil, errFakeDB }

func (r *fakeRowsAffErrResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeRowsAffErrResult) RowsAffected() (int64, error) { return 0, errFakeDB }

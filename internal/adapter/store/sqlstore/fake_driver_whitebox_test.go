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
	sql.Register("fake-tx-rows-err", &fakeTxRowsErrDriver{})
	sql.Register("fake-tx-scan-err", &fakeTxScanErrDriver{})
	// Transaction-level drivers: Begin succeeds, Query returns 0 rows.
	sql.Register("fake-tx-rows-aff-err", &fakeTxRowsAffErrDriver{})
	sql.Register("fake-tx-first-exec-fail", &fakeTxFirstExecFailDriver{})
	sql.Register("fake-tx-exec-ok-commit-fail", &fakeTxExecOKCommitFailDriver{})
	// Transaction-level drivers: Begin succeeds, Query returns 1 row ("wl-fake").
	sql.Register("fake-tx-with-row-first-exec-fail", &fakeTxWithRowFirstExecFailDriver{})
	sql.Register("fake-tx-with-row-exec-fail-after-1", &fakeTxWithRowExecFailAfter1Driver{})
	// Transaction-level drivers: Begin succeeds, Query returns 0 rows, first 2 execs ok, 3rd fails.
	sql.Register("fake-tx-exec-fail-after-2", &fakeTxExecFailAfter2Driver{})
	// Accept-flow drivers: Begin succeeds; QueryRow 1 = pending transfer, QueryRow 2 = active item.
	// Execs succeed up to failOnExec-1, then fail at failOnExec. commitFail=true fails commit instead.
	sql.Register("fake-tx-accept-exec-fail-at-1", &fakeAcceptTxDriver{failOnExec: 1})
	sql.Register("fake-tx-accept-exec-fail-at-2", &fakeAcceptTxDriver{failOnExec: 2})
	sql.Register("fake-tx-accept-exec-fail-at-3", &fakeAcceptTxDriver{failOnExec: 3})
	sql.Register("fake-tx-accept-exec-fail-at-4", &fakeAcceptTxDriver{failOnExec: 4})
	sql.Register("fake-tx-accept-exec-fail-at-5", &fakeAcceptTxDriver{failOnExec: 5})
	sql.Register("fake-tx-accept-commit-fail", &fakeAcceptTxDriver{commitFail: true})
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
func (c *fakeRowsErrConn) Close() error              { return nil }
func (c *fakeRowsErrConn) Begin() (driver.Tx, error) { return nil, errFakeDB }

func (s *fakeRowsErrStmt) Close() error                                 { return nil }
func (s *fakeRowsErrStmt) NumInput() int                                { return -1 }
func (s *fakeRowsErrStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, errFakeDB }
func (s *fakeRowsErrStmt) Query(_ []driver.Value) (driver.Rows, error)  { return &fakeErrRows{}, nil }

func (r *fakeErrRows) Columns() []string           { return nil }
func (r *fakeErrRows) Close() error                { return nil }
func (r *fakeErrRows) Next(_ []driver.Value) error { return errFakeDB }

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
func (c *fakeScanErrConn) Close() error              { return nil }
func (c *fakeScanErrConn) Begin() (driver.Tx, error) { return nil, errFakeDB }

func (s *fakeScanErrStmt) Close() error                                 { return nil }
func (s *fakeScanErrStmt) NumInput() int                                { return -1 }
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
func (c *fakeCommitErrConn) Close() error                          { return nil }
func (c *fakeCommitErrConn) Begin() (driver.Tx, error)             { return &fakeCommitErrTx{}, nil }

func (tx *fakeCommitErrTx) Commit() error   { return errFakeDB }
func (tx *fakeCommitErrTx) Rollback() error { return nil }

func (s *fakeOKStmt) Close() error                                 { return nil }
func (s *fakeOKStmt) NumInput() int                                { return -1 }
func (s *fakeOKStmt) Exec(_ []driver.Value) (driver.Result, error) { return &fakeOKResult{}, nil }
func (s *fakeOKStmt) Query(_ []driver.Value) (driver.Rows, error)  { return nil, errFakeDB }

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
func (c *fakeRowsAffErrConn) Close() error              { return nil }
func (c *fakeRowsAffErrConn) Begin() (driver.Tx, error) { return nil, errFakeDB }

func (s *fakeRowsAffErrStmt) Close() error  { return nil }
func (s *fakeRowsAffErrStmt) NumInput() int { return -1 }
func (s *fakeRowsAffErrStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeRowsAffErrResult{}, nil
}
func (s *fakeRowsAffErrStmt) Query(_ []driver.Value) (driver.Rows, error) { return nil, errFakeDB }

func (r *fakeRowsAffErrResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeRowsAffErrResult) RowsAffected() (int64, error) { return 0, errFakeDB }

// ── fake-tx-rows-err ──────────────────────────────────────────────────────────
// Begin() succeeds; Query returns rows whose Next() returns errFakeDB (rows.Err).

type (
	fakeTxRowsErrDriver struct{}
	fakeTxRowsErrConn   struct{}
	fakeTxRowsErrTx     struct{}
	fakeTxRowsErrStmt   struct{}
)

func (d *fakeTxRowsErrDriver) Open(_ string) (driver.Conn, error) {
	return &fakeTxRowsErrConn{}, nil
}
func (c *fakeTxRowsErrConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxRowsErrStmt{}, nil
}
func (c *fakeTxRowsErrConn) Close() error              { return nil }
func (c *fakeTxRowsErrConn) Begin() (driver.Tx, error) { return &fakeTxRowsErrTx{}, nil }

func (tx *fakeTxRowsErrTx) Commit() error   { return nil }
func (tx *fakeTxRowsErrTx) Rollback() error { return nil }

func (s *fakeTxRowsErrStmt) Close() error  { return nil }
func (s *fakeTxRowsErrStmt) NumInput() int { return -1 }
func (s *fakeTxRowsErrStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeOKResult{}, nil
}
func (s *fakeTxRowsErrStmt) Query(_ []driver.Value) (driver.Rows, error) { return &fakeErrRows{}, nil }

// ── fake-tx-scan-err ──────────────────────────────────────────────────────────
// Begin() succeeds; Query returns rows where Scan fails.

type (
	fakeTxScanErrDriver struct{}
	fakeTxScanErrConn   struct{}
	fakeTxScanErrTx     struct{}
	fakeTxScanErrStmt   struct{}
)

func (d *fakeTxScanErrDriver) Open(_ string) (driver.Conn, error) {
	return &fakeTxScanErrConn{}, nil
}
func (c *fakeTxScanErrConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxScanErrStmt{}, nil
}
func (c *fakeTxScanErrConn) Close() error              { return nil }
func (c *fakeTxScanErrConn) Begin() (driver.Tx, error) { return &fakeTxScanErrTx{}, nil }

func (tx *fakeTxScanErrTx) Commit() error   { return nil }
func (tx *fakeTxScanErrTx) Rollback() error { return nil }

func (s *fakeTxScanErrStmt) Close() error  { return nil }
func (s *fakeTxScanErrStmt) NumInput() int { return -1 }
func (s *fakeTxScanErrStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeOKResult{}, nil
}
func (s *fakeTxScanErrStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeScanErrRows{}, nil
}

// ── fake-tx-rows-aff-err ──────────────────────────────────────────────────────
// Begin succeeds; Query returns 0 rows; Exec returns a result where RowsAffected() fails.

type (
	fakeTxRowsAffErrDriver struct{}
	fakeTxRowsAffErrConn   struct{}
	fakeTxRowsAffErrTx     struct{}
	fakeTxRowsAffErrStmt   struct{}
)

func (d *fakeTxRowsAffErrDriver) Open(_ string) (driver.Conn, error) {
	return &fakeTxRowsAffErrConn{}, nil
}
func (c *fakeTxRowsAffErrConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxRowsAffErrStmt{}, nil
}
func (c *fakeTxRowsAffErrConn) Close() error              { return nil }
func (c *fakeTxRowsAffErrConn) Begin() (driver.Tx, error) { return &fakeTxRowsAffErrTx{}, nil }
func (tx *fakeTxRowsAffErrTx) Commit() error              { return nil }
func (tx *fakeTxRowsAffErrTx) Rollback() error            { return nil }
func (s *fakeTxRowsAffErrStmt) Close() error              { return nil }
func (s *fakeTxRowsAffErrStmt) NumInput() int             { return -1 }
func (s *fakeTxRowsAffErrStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeRowsAffErrResult{}, nil
}
func (s *fakeTxRowsAffErrStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeEmptyRows{}, nil
}

// ── fake-tx-first-exec-fail ───────────────────────────────────────────────────
// Begin succeeds; Query returns 0 rows; first ExecContext fails.

type (
	fakeTxFirstExecFailDriver struct{}
	fakeTxFirstExecFailConn   struct{}
	fakeTxFirstExecFailTx     struct{}
	fakeTxFirstExecFailStmt   struct{}
	fakeEmptyRows             struct{}
)

func (d *fakeTxFirstExecFailDriver) Open(_ string) (driver.Conn, error) {
	return &fakeTxFirstExecFailConn{}, nil
}
func (c *fakeTxFirstExecFailConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxFirstExecFailStmt{}, nil
}
func (c *fakeTxFirstExecFailConn) Close() error              { return nil }
func (c *fakeTxFirstExecFailConn) Begin() (driver.Tx, error) { return &fakeTxFirstExecFailTx{}, nil }
func (tx *fakeTxFirstExecFailTx) Commit() error              { return nil }
func (tx *fakeTxFirstExecFailTx) Rollback() error            { return nil }
func (s *fakeTxFirstExecFailStmt) Close() error              { return nil }
func (s *fakeTxFirstExecFailStmt) NumInput() int             { return -1 }
func (s *fakeTxFirstExecFailStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return nil, errFakeDB
}
func (s *fakeTxFirstExecFailStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeEmptyRows{}, nil
}
func (r *fakeEmptyRows) Columns() []string           { return []string{"col"} }
func (r *fakeEmptyRows) Close() error                { return nil }
func (r *fakeEmptyRows) Next(_ []driver.Value) error { return io.EOF }

// ── fake-tx-exec-ok-commit-fail ──────────────────────────────────────────────
// Begin succeeds; Query returns 0 rows; all Execs succeed; Commit fails.

type (
	fakeTxExecOKCommitFailDriver struct{}
	fakeTxExecOKCommitFailConn   struct{}
	fakeTxExecOKCommitFailTx     struct{}
	fakeTxExecOKCommitFailStmt   struct{}
)

func (d *fakeTxExecOKCommitFailDriver) Open(_ string) (driver.Conn, error) {
	return &fakeTxExecOKCommitFailConn{}, nil
}
func (c *fakeTxExecOKCommitFailConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxExecOKCommitFailStmt{}, nil
}
func (c *fakeTxExecOKCommitFailConn) Close() error { return nil }
func (c *fakeTxExecOKCommitFailConn) Begin() (driver.Tx, error) {
	return &fakeTxExecOKCommitFailTx{}, nil
}
func (tx *fakeTxExecOKCommitFailTx) Commit() error   { return errFakeDB }
func (tx *fakeTxExecOKCommitFailTx) Rollback() error { return nil }
func (s *fakeTxExecOKCommitFailStmt) Close() error   { return nil }
func (s *fakeTxExecOKCommitFailStmt) NumInput() int  { return -1 }
func (s *fakeTxExecOKCommitFailStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeOKResult{}, nil
}
func (s *fakeTxExecOKCommitFailStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeEmptyRows{}, nil
}

// ── fake-tx-with-row-first-exec-fail ─────────────────────────────────────────
// Begin succeeds; Query returns 1 row ("wl-fake"); first ExecContext fails.

type (
	fakeTxWithRowFirstExecFailDriver struct{}
	fakeTxWithRowFirstExecFailConn   struct{}
	fakeTxWithRowFirstExecFailTx     struct{}
	fakeTxWithRowFirstExecFailStmt   struct{}
	fakeOneRow                       struct{ done bool }
)

func (d *fakeTxWithRowFirstExecFailDriver) Open(_ string) (driver.Conn, error) {
	return &fakeTxWithRowFirstExecFailConn{}, nil
}
func (c *fakeTxWithRowFirstExecFailConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxWithRowFirstExecFailStmt{}, nil
}
func (c *fakeTxWithRowFirstExecFailConn) Close() error { return nil }
func (c *fakeTxWithRowFirstExecFailConn) Begin() (driver.Tx, error) {
	return &fakeTxWithRowFirstExecFailTx{}, nil
}
func (tx *fakeTxWithRowFirstExecFailTx) Commit() error   { return nil }
func (tx *fakeTxWithRowFirstExecFailTx) Rollback() error { return nil }
func (s *fakeTxWithRowFirstExecFailStmt) Close() error   { return nil }
func (s *fakeTxWithRowFirstExecFailStmt) NumInput() int  { return -1 }
func (s *fakeTxWithRowFirstExecFailStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return nil, errFakeDB
}
func (s *fakeTxWithRowFirstExecFailStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeOneRow{}, nil // fresh instance per call; done resets automatically
}
func (r *fakeOneRow) Columns() []string { return []string{"wear_log_id"} }
func (r *fakeOneRow) Close() error      { return nil }
func (r *fakeOneRow) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = "wl-fake"
	return nil
}

// ── fake-tx-with-row-exec-fail-after-1 ───────────────────────────────────────
// Begin succeeds; Query returns 1 row; first Exec succeeds; second Exec fails.
// The exec counter is on the connection so it persists across prepared statements.

type (
	fakeTxWithRowExecFailAfter1Driver struct{}
	fakeTxWithRowExecFailAfter1Conn   struct{ execCount int }
	fakeTxWithRowExecFailAfter1Tx     struct{}
	fakeTxWithRowExecFailAfter1Stmt   struct {
		conn *fakeTxWithRowExecFailAfter1Conn
	}
)

func (d *fakeTxWithRowExecFailAfter1Driver) Open(_ string) (driver.Conn, error) {
	return &fakeTxWithRowExecFailAfter1Conn{}, nil
}
func (c *fakeTxWithRowExecFailAfter1Conn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxWithRowExecFailAfter1Stmt{conn: c}, nil
}
func (c *fakeTxWithRowExecFailAfter1Conn) Close() error { return nil }
func (c *fakeTxWithRowExecFailAfter1Conn) Begin() (driver.Tx, error) {
	return &fakeTxWithRowExecFailAfter1Tx{}, nil
}
func (tx *fakeTxWithRowExecFailAfter1Tx) Commit() error   { return nil }
func (tx *fakeTxWithRowExecFailAfter1Tx) Rollback() error { return nil }
func (s *fakeTxWithRowExecFailAfter1Stmt) Close() error   { return nil }
func (s *fakeTxWithRowExecFailAfter1Stmt) NumInput() int  { return -1 }
func (s *fakeTxWithRowExecFailAfter1Stmt) Exec(_ []driver.Value) (driver.Result, error) {
	s.conn.execCount++
	if s.conn.execCount > 1 {
		return nil, errFakeDB
	}
	return &fakeOKResult{}, nil
}
func (s *fakeTxWithRowExecFailAfter1Stmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeOneRow{}, nil // fresh instance per call; done resets automatically
}

// ── fake-tx-exec-fail-after-2 ────────────────────────────────────────────────
// Begin succeeds; Query returns 0 rows; first 2 Execs succeed; 3rd fails.
// The exec counter is on the connection so it persists across prepared statements.

type (
	fakeTxExecFailAfter2Driver struct{}
	fakeTxExecFailAfter2Conn   struct{ execCount int }
	fakeTxExecFailAfter2Tx     struct{}
	fakeTxExecFailAfter2Stmt   struct{ conn *fakeTxExecFailAfter2Conn }
)

func (d *fakeTxExecFailAfter2Driver) Open(_ string) (driver.Conn, error) {
	return &fakeTxExecFailAfter2Conn{}, nil
}
func (c *fakeTxExecFailAfter2Conn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeTxExecFailAfter2Stmt{conn: c}, nil
}
func (c *fakeTxExecFailAfter2Conn) Close() error              { return nil }
func (c *fakeTxExecFailAfter2Conn) Begin() (driver.Tx, error) { return &fakeTxExecFailAfter2Tx{}, nil }
func (tx *fakeTxExecFailAfter2Tx) Commit() error              { return nil }
func (tx *fakeTxExecFailAfter2Tx) Rollback() error            { return nil }
func (s *fakeTxExecFailAfter2Stmt) Close() error              { return nil }
func (s *fakeTxExecFailAfter2Stmt) NumInput() int             { return -1 }
func (s *fakeTxExecFailAfter2Stmt) Exec(_ []driver.Value) (driver.Result, error) {
	s.conn.execCount++
	if s.conn.execCount > 2 {
		return nil, errFakeDB
	}
	return &fakeOKResult{}, nil
}
func (s *fakeTxExecFailAfter2Stmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &fakeEmptyRows{}, nil
}

// ── fake-tx-accept-exec-fail-at-N / fake-tx-accept-commit-fail ───────────────
// Begin succeeds; first QueryRow returns a valid pending-transfer row (6 cols);
// second QueryRow returns a valid active-item row (3 cols, owner = "sender-fake");
// ExecContext calls succeed until the Nth (failOnExec > 0), then fail.
// If commitFail is true, all execs succeed but Commit returns errFakeDB.

type fakeAcceptTxDriver struct {
	failOnExec int
	commitFail bool
}
type fakeAcceptTxConn struct {
	queryCount int
	execCount  int
	d          *fakeAcceptTxDriver
}
type fakeAcceptTxTx struct{ conn *fakeAcceptTxConn }
type fakeAcceptTxStmt struct{ conn *fakeAcceptTxConn }

func (d *fakeAcceptTxDriver) Open(_ string) (driver.Conn, error) {
	return &fakeAcceptTxConn{d: d}, nil
}
func (c *fakeAcceptTxConn) Prepare(_ string) (driver.Stmt, error) {
	return &fakeAcceptTxStmt{conn: c}, nil
}
func (c *fakeAcceptTxConn) Close() error              { return nil }
func (c *fakeAcceptTxConn) Begin() (driver.Tx, error) { return &fakeAcceptTxTx{conn: c}, nil }
func (tx *fakeAcceptTxTx) Commit() error {
	if tx.conn.d.commitFail {
		return errFakeDB
	}
	return nil
}
func (tx *fakeAcceptTxTx) Rollback() error { return nil }
func (s *fakeAcceptTxStmt) Close() error   { return nil }
func (s *fakeAcceptTxStmt) NumInput() int  { return -1 }
func (s *fakeAcceptTxStmt) Exec(_ []driver.Value) (driver.Result, error) {
	s.conn.execCount++
	if s.conn.d.failOnExec > 0 && s.conn.execCount == s.conn.d.failOnExec {
		return nil, errFakeDB
	}
	return &fakeOKResult{}, nil
}
func (s *fakeAcceptTxStmt) Query(_ []driver.Value) (driver.Rows, error) {
	s.conn.queryCount++
	switch s.conn.queryCount {
	case 1:
		return &fakeAcceptTransferRow{}, nil
	case 2:
		return &fakeAcceptItemRow{}, nil
	default:
		return &fakeEmptyRows{}, nil
	}
}

// fakeAcceptTransferRow returns one row: (id, item_id, sender_id, recipient_id, "pending", 0, created_at).
type fakeAcceptTransferRow struct{ done bool }

func (r *fakeAcceptTransferRow) Columns() []string {
	return []string{"id", "item_id", "sender_id", "recipient_id", "status", "transfer_history", "created_at"}
}
func (r *fakeAcceptTransferRow) Close() error { return nil }
func (r *fakeAcceptTransferRow) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = "tr-fake"
	dest[1] = "item-fake"
	dest[2] = "sender-fake"
	dest[3] = "recip-fake"
	dest[4] = "pending"
	dest[5] = int64(0)
	dest[6] = "2025-01-01T00:00:00Z"
	return nil
}

// fakeAcceptItemRow returns one row: (owner_id="sender-fake", archived_at=nil, disposal_reason=nil).
type fakeAcceptItemRow struct{ done bool }

func (r *fakeAcceptItemRow) Columns() []string {
	return []string{"owner_id", "archived_at", "disposal_reason"}
}
func (r *fakeAcceptItemRow) Close() error { return nil }
func (r *fakeAcceptItemRow) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = "sender-fake"
	dest[1] = nil
	dest[2] = nil
	return nil
}

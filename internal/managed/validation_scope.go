package managed

import (
	"context"
	"database/sql"
	"errors"
)

// A validation traversal repeatedly reaches the same predecessor journals.
// Memoize only inside one synchronous, read-only traversal of a SQL transaction.
// Never attach the memo to a store/transaction: another call after a write must
// inspect the changed rows. Ordinary DB queries first acquire a read transaction
// for this traversal only. The wrapper exposes no mutation or transaction API.
type validationSnapshot struct {
	querier
	completed map[string]error
	visiting  map[string]bool
	recovery  *projectRecoveryReadView
}

func validateSnapshot(q querier, name string, check func(querier) error) error {
	if db, ok := q.(*sql.DB); ok {
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return err
		}
		defer tx.Rollback()
		return validateSnapshot(tx, name, check)
	}
	scope, ok := q.(*validationSnapshot)
	if !ok {
		if _, snapshot := q.(*sql.Tx); !snapshot {
			return check(q)
		}
		scope = &validationSnapshot{querier: q, completed: map[string]error{}, visiting: map[string]bool{}}
	}
	if err, done := scope.completed[name]; done {
		return err
	}
	if scope.visiting[name] {
		return errors.New("cyclic project validation dependency")
	}
	scope.visiting[name] = true
	err := check(scope)
	delete(scope.visiting, name)
	scope.completed[name] = err
	return err
}

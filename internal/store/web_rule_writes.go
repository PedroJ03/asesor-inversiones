package store

import "fmt"

// SetRuleEnabled updates the enabled flag of an alert rule without changing
// its armed/triggered state. It is a platform-owned additive write kept in a
// separate file so the frozen store contract in store.go remains untouched.
func (s *Store) SetRuleEnabled(id int64, enabled bool) error {
	res, err := s.db.Exec("UPDATE alert_rules SET enabled = ? WHERE id = ?", enabled, id)
	if err != nil {
		return fmt.Errorf("set rule enabled: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: rule %d not found", ErrInvalidRule, id)
	}
	return nil
}

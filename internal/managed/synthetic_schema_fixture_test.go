package managed

import "testing"

// Legacy focused tests sometimes construct synthetic compatibility states with
// today's encoder. They are not original-writer migration evidence. Explicitly
// omit the empty successor app tables before assigning an older synthetic marker;
// otherwise the production reader must correctly refuse that mixed schema.
// This helper is test-only and refuses any app history or namespace reservation.
// Original-writer fixtures never use it; user stores must never be relabeled.
func omitEmptyAppSuccessorForSyntheticFixture(t *testing.T, s *store) {
	t.Helper()
	for _, table := range grubberyTables {
		var count int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM "+table.name).Scan(&count))
		expected := 0
		if table.name == "grubbery_app_clock" {
			expected = 1
			var ordinal int
			g02Check(t, s.db.QueryRow("SELECT ordinal FROM grubbery_app_clock WHERE singleton=1").Scan(&ordinal))
			if ordinal != 0 {
				t.Fatal("synthetic fixture has app history")
			}
		}
		if count != expected {
			t.Fatal("synthetic fixture has app history/reservations")
		}
	}
	for i := len(grubberyTables) - 1; i >= 0; i-- {
		_, err := s.db.Exec("DROP TABLE " + grubberyTables[i].name)
		g02Check(t, err)
	}
}

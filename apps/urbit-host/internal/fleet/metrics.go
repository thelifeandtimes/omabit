package fleet

import (
	"strconv"
	"strings"
)

func parseMemory(s string) uint64 {
	f := strings.Fields(strings.TrimSpace(strings.Split(s, "/")[0]))
	if len(f) == 0 {
		return 0
	}
	v := f[0]
	units := []struct {
		s string
		m float64
	}{{"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"B", 1}}
	for _, u := range units {
		if strings.HasSuffix(v, u.s) {
			x, _ := strconv.ParseFloat(strings.TrimSuffix(v, u.s), 64)
			return uint64(x * u.m)
		}
	}
	return 0
}
func BudgetMetrics(stats []map[string]string, c HostConfig) map[string]any {
	var cpu float64
	var mem uint64
	a := []map[string]any{}
	for _, s := range stats {
		x, _ := strconv.ParseFloat(strings.TrimSuffix(s["CPUPerc"], "%"), 64)
		n := parseMemory(s["MemUsage"])
		cpu += x
		mem += n
		a = append(a, map[string]any{"container": s["Name"], "cpu_percent": x, "memory_bytes": n})
	}
	warnings := []string{}
	if c.CPUWarning > 0 && cpu > c.CPUWarning {
		warnings = append(warnings, "shared fleet CPU warning threshold exceeded")
	}
	if c.MemoryWarning > 0 && mem > c.MemoryWarning {
		warnings = append(warnings, "shared fleet memory warning threshold exceeded")
	}
	return map[string]any{"observed_at": Now(), "instances": a, "fleet_cpu_percent": cpu, "fleet_memory_bytes": mem, "warnings": warnings, "hard_limits": false}
}

package hostcompat

import "time"

func freshRelease(b []byte) bool {
	m, err := parseControl(string(b))
	if err != nil || len(m) != 1 {
		return false
	}
	now := time.Now()
	date, err := time.Parse(time.RFC1123Z, m[0]["Date"])
	if err != nil {
		date, err = time.Parse(time.RFC1123, m[0]["Date"])
	}
	if err != nil || date.After(now.Add(10*time.Minute)) {
		return false
	}
	if s := m[0]["Valid-Until"]; s != "" {
		until, err := time.Parse(time.RFC1123Z, s)
		if err != nil {
			until, err = time.Parse(time.RFC1123, s)
		}
		if err != nil || !now.Before(until) || until.Before(date) {
			return false
		}
	}
	return true
}

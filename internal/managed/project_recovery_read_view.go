package managed

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"

// A view belongs to one read-only validation call. It must never survive a write,
// be stored on store/Server, or cache engine observations. Nested validators share
// already verified database history; each fresh admission/transition makes a new
// view. The querier interface deliberately exposes no mutation methods.
type projectRecoveryReadView struct {
	querier
	histories map[projectRecoveryHistoryKey]projectRecoveryHistoryValue
}
type projectRecoveryHistoryKey struct {
	project string
	current bool
}
type projectRecoveryHistoryValue struct {
	history            projectnetwork.AnchorRecoveryHistory
	originals, current []string
}

func projectRecoveryReadOnly(q querier) querier {
	if scope, ok := q.(*validationSnapshot); ok {
		if scope.recovery == nil {
			scope.recovery = &projectRecoveryReadView{querier: q, histories: map[projectRecoveryHistoryKey]projectRecoveryHistoryValue{}}
		}
		return q
	}
	if _, ok := q.(*projectRecoveryReadView); ok {
		return q
	}
	return &projectRecoveryReadView{querier: q, histories: map[projectRecoveryHistoryKey]projectRecoveryHistoryValue{}}
}
func projectRecoveryHistory(q querier, project string, current bool) (projectnetwork.AnchorRecoveryHistory, []string, []string, error) {
	v, ok := q.(*projectRecoveryReadView)
	if scope, scoped := q.(*validationSnapshot); scoped && scope.recovery != nil {
		v, ok = scope.recovery, true
	}
	if !ok {
		return readProjectRecoveryHistory(q, project, current)
	}
	key := projectRecoveryHistoryKey{project, current}
	if cached, hit := v.histories[key]; hit {
		return cached.history.Clone(), append([]string(nil), cached.originals...), append([]string(nil), cached.current...), nil
	}
	h, originals, ids, err := readProjectRecoveryHistory(q, project, current)
	if err != nil {
		return h, originals, ids, err
	}
	v.histories[key] = projectRecoveryHistoryValue{history: h.Clone(), originals: append([]string(nil), originals...), current: append([]string(nil), ids...)}
	return h, originals, ids, nil
}

package projectnetwork

import (
	"reflect"
	"time"
)

// equalEvidence compares acyclic project model values without giving time.Time's
// in-memory timezone or monotonic metadata authority over persisted evidence.
// JSON preserves the instant, but not that metadata. All other fields, including
// nil versus empty slices, retain exact equality. Inputs are never rewritten.
func equalEvidence(a, b any) bool {
	return equalEvidenceValue(reflect.ValueOf(a), reflect.ValueOf(b))
}

var evidenceTimeType = reflect.TypeOf(time.Time{})

func equalEvidenceValue(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}
	if a.Type() == evidenceTimeType {
		return a.Interface().(time.Time).UTC() == b.Interface().(time.Time).UTC()
	}
	switch a.Kind() {
	case reflect.Interface, reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return equalEvidenceValue(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !equalEvidenceValue(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		if a.Kind() == reflect.Slice && a.IsNil() != b.IsNil() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !equalEvidenceValue(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	default:
		// Project evidence currently consists of exported structs, slices, pointers
		// and scalar fields. Keep other types exact rather than weakening comparison.
		if !a.CanInterface() || !b.CanInterface() {
			return false
		}
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

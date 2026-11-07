package deskwork

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func ListDesks(ship string) (LensRequest, error) {
	if !validSelection(ship, "base") {
		return LensRequest{}, errors.New("invalid ship selection")
	}
	expression := fmt.Sprintf(`=/  wen  (scot %%da now)
=/  desks  ~(tap in .^((set @tas) %%cd /%s//[wen]))
?>  (lte (lent desks) 8.192)
`+"`@ux`"+`(en:json:html [%%a (turn desks |=(d=@tas [%%s d]))])`, ship)
	return LensRequest{expression: expression}, nil
}

// FilePaths returns Clay path segments at an immutable revision. Conversion to
// Unix names is checked separately; no runtime-supplied path is opened directly.
func FilePaths(ship, desk string, revision uint64) (LensRequest, error) {
	if !validSelection(ship, desk) || revision == 0 || revision > 1<<53-1 {
		return LensRequest{}, errors.New("invalid desk revision")
	}
	expression := fmt.Sprintf(`=/  wen  (scot %%da now)
?>  (~(has in .^((set @tas) %%cd /%s//[wen])) %%%s)
?>  (lte %s let:.^(domo:clay %%cv /%s/%s/[wen]))
=/  paths  .^((list path) %%ct /%s/%s/%s)
?>  (lte (lent paths) 8.192)
`+"`@ux`"+`(en:json:html [%%a (turn paths |=(p=path [%%a (turn p |=(s=@ta [%%s s]))]))])`, ship, desk, hoonDecimal(revision), ship, desk, ship, desk, hoonDecimal(revision))
	return LensRequest{expression: expression}, nil
}

// AutocommitActive reads actual Behn timers. Kiln's saved timer timestamp remains
// populated after cancellation, so it is not evidence of active autocommit.
// A timer for any mount conservatively prevents ungated source writes.
func AutocommitActive(ship string) (LensRequest, error) {
	if !validSelection(ship, "base") {
		return LensRequest{}, errors.New("invalid ship selection")
	}
	expression := fmt.Sprintf(`=/  timers  .^((list [@da duct]) %%bx /%s//(scot %%da now)/debug/timers)
%%+  lien  timers
|=  t=[@da duct]
%%+  lien  +.t
|=  p=path
=/  reversed=path  (flop p)
?&(?=([%%gall %%use %%hood *] p) ?=([%%autocommit %%kiln *] reversed))`, ship)
	return LensRequest{expression: expression}, nil
}
func UnixPaths(clay [][]string) ([]string, error) {
	if clay == nil || len(clay) > MaxFiles {
		return nil, errors.New("invalid Clay path list")
	}
	paths := make([]string, 0, len(clay))
	seen := map[string]bool{}
	for _, p := range clay {
		if len(p) < 2 || len(p) > 17 {
			return nil, errors.New("unsupported Clay path")
		}
		for _, s := range p {
			if !segment.MatchString(s) {
				return nil, errors.New("unsupported Clay path segment")
			}
		}
		name := strings.Join(p[:len(p)-1], "/") + "." + p[len(p)-1]
		if !ValidPath(name) || seen[name] {
			return nil, errors.New("invalid or duplicate Clay path")
		}
		seen[name] = true
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		for cut := strings.LastIndex(name, "/"); cut >= 0; cut = strings.LastIndex(name[:cut], "/") {
			if seen[name[:cut]] {
				return nil, errors.New("Clay file and directory paths collide")
			}
		}
	}
	return paths, nil
}

const MaxDesks = 2048

func NormalizeDeskNames(names []string) ([]string, error) {
	if names == nil || len(names) > MaxDesks {
		return nil, errors.New("invalid desk inventory")
	}
	out := append([]string{}, names...)
	sort.Strings(out)
	for i, name := range out {
		if !deskName.MatchString(name) || i > 0 && out[i-1] == name {
			return nil, errors.New("invalid or duplicate desk name")
		}
	}
	return out, nil
}

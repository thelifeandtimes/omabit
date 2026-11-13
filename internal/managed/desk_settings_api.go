package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"io"
	"net/http"
	"time"
)

func (r *Server) serveDeskSettings(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost && req.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if req.Method == http.MethodPut && role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "editing workflow settings requires operator access"}})
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, 16<<10)
	d := json.NewDecoder(req.Body)
	d.DisallowUnknownFields()
	var value deskwork.Setting
	var err error
	if req.Method == http.MethodPost {
		var target apppolicy.Target
		if d.Decode(&target) != nil || d.Decode(new(any)) != io.EOF {
			r.respondError(w, bad("invalid workflow setting selection"))
			return
		}
		value, err = r.s.deskSetting(target)
	} else {
		var update deskwork.SettingUpdate
		if d.Decode(&update) != nil || d.Decode(new(any)) != io.EOF {
			r.respondError(w, bad("invalid workflow setting update"))
			return
		}
		value, err = r.s.putDeskSetting(actor, update)
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, protocol.DeskSetting{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, ReadAt: time.Now().UTC(), Setting: value})
}

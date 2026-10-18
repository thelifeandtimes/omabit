package managed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const managementFenceKey = "management_update_v1"

// The released token is retained for lost-acknowledgement replay. This record
// lives in the existing meta table; old runners must be schema-fenced by release
// integration because they do not understand its admission semantics.
type managementFence struct {
	Token string `json:"token"`
	Held  bool   `json:"held"`
}

func updateDigest(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == s
}

func loadManagementFence(s *store) (managementFence, error) {
	var f managementFence
	var data string
	err := s.db.QueryRow("SELECT value FROM meta WHERE key=?", managementFenceKey).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if json.Unmarshal([]byte(data), &f) != nil || !updateDigest(f.Token) {
		return f, errors.New("invalid management update fence; preserve state and review activation journal")
	}
	return f, nil
}

func (r *Server) saveManagementFence(f managementFence) error {
	data, err := json.Marshal(f)
	if err == nil {
		_, err = r.s.db.Exec("INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", managementFenceKey, string(data))
	}
	if err == nil {
		r.updateFence = f
	}
	return err
}

func executableDigest() (string, error) {
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Caller holds r.mu, shared with all API, scheduler, startup and bulk admissions.
// Refuse busy work without canceling it. An idle successful acquire commits the
// fence before acknowledging; no TTL or restart can silently reopen admission.
func (r *Server) serveManagementUpdate(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() {
		r.respondError(w, conflict("runner_unavailable", "runner is stopping or journal requires review"))
		return
	}
	if req.Method != http.MethodGet {
		var body protocol.ManagementUpdateRequest
		if req.Method != http.MethodPost {
			r.respondError(w, bad("require GET or POST"))
			return
		}
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		if body.RunnerID != r.s.runnerID || !updateDigest(body.Token) {
			r.respondError(w, bad("require exact runner identity and 256-bit update token"))
			return
		}
		if r.updateFence.Held && r.updateFence.Token != body.Token {
			r.respondError(w, conflict("update_token_conflict", "another activation owns the durable fence"))
			return
		}
		switch body.Action {
		case "acquire":
			if r.updateFence.Token == body.Token && !r.updateFence.Held {
				r.respondError(w, conflict("update_token_released", "use a new token for a new activation"))
				return
			}
			all, err := instances(r.s.db)
			if err != nil {
				r.respondError(w, err)
				return
			}
			busy := r.updateBulk != 0 || r.updateWorkers != 0
			for _, v := range all {
				busy = busy || v.Public.ActiveOperationID != ""
			}
			if busy {
				r.respondError(w, conflict("update_busy", "accepted jobs or bulk transfers are active; let them finish and retry; no work was canceled"))
				return
			}
			if err := r.saveManagementFence(managementFence{Token: body.Token, Held: true}); err != nil {
				r.respondError(w, err)
				return
			}
		case "release":
			if r.updateFence.Token != body.Token || body.ExpectedSHA256 != r.executableSHA256 {
				r.respondError(w, conflict("update_activation_mismatch", "release requires retained token and verified executable generation"))
				return
			}
			if r.updateFence.Held {
				if err := r.saveManagementFence(managementFence{Token: body.Token}); err != nil {
					r.respondError(w, err)
					return
				}
			}
		default:
			r.respondError(w, bad("unknown management update action"))
			return
		}
	}
	respond(w, 200, protocol.ManagementUpdateStatus{RunnerID: r.s.runnerID, Mode: r.s.mode, Quiesced: r.updateFence.Held, Token: r.updateFence.Token, StateSchema: StateSchemaVersion, MinStateSchema: 11, PID: os.Getpid(), ExecutableSHA256: r.executableSHA256})
}

func (r *Server) beginUpdateBulk() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() {
		return conflict("runner_unavailable", "runner is unavailable")
	}
	if r.updateFence.Held {
		return conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	r.updateBulk++
	r.wg.Add(1)
	return nil
}

func (r *Server) endUpdateBulk() {
	r.mu.Lock()
	r.updateBulk--
	r.mu.Unlock()
	r.wg.Done()
}

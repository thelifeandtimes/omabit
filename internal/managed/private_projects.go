package managed

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const privateProjectRegistrationKind = privatechain.RegistrationKind

// Registration reserves an immutable reviewed contract, never keys or runtime
// resources. It deliberately cannot enter the existing fake-project lifecycle.
type privateProjectRegistration struct {
	RequestID    string            `json:"request_id"`
	Plan         privatechain.Plan `json:"plan"`
	ReviewDigest string            `json:"review_digest"`
	ReplayOnly   bool              `json:"-"`
}
type privateProjectMember struct {
	privatechain.Member
	RuntimeID string `json:"runtime_id"`
}
type storedPrivateProject struct {
	Version                 int                    `json:"version"`
	Binding                 privatechain.Binding   `json:"binding"`
	Revision                uint64                 `json:"revision"`
	Phase                   string                 `json:"phase"`
	Plan                    privatechain.Plan      `json:"plan"`
	Members                 []privateProjectMember `json:"members"`
	RegistrationOperationID string                 `json:"registration_operation_id"`
	CreatedAt               time.Time              `json:"created_at"`
}

func initializePrivateProjects(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS private_projects(id TEXT PRIMARY KEY,body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS private_project_members(instance_id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES private_projects(id),http_port INTEGER NOT NULL UNIQUE CHECK(http_port BETWEEN 1024 AND 65535),runtime_id TEXT NOT NULL UNIQUE);`)
	return err
}

func privateProjectFingerprint(req privateProjectRegistration) string {
	data, _ := json.Marshal(struct {
		Kind    string
		Request privateProjectRegistration
	}{privateProjectRegistrationKind, req})
	return string(data)
}

// projectReservations is the common allocation fence for fake/private projects
// and standalone ships. Both representations must agree; partial state refuses.
func projectReservations(q querier) (projectnetwork.Reservations, error) {
	reserved, err := fakeProjectReservations(q)
	if err != nil {
		return reserved, err
	}
	reserved, err = privateProjectReservations(q, reserved)
	if err != nil {
		return reserved, err
	}
	if err = expansionReservations(q, &reserved); err != nil {
		return reserved, err
	}
	return reserved, validatePrivateMemberLaunches(q)
}

func privateProjectReservations(q querier, reserved projectnetwork.Reservations) (projectnetwork.Reservations, error) {
	fail := func() (projectnetwork.Reservations, error) {
		return reserved, errors.New("invalid private-chain project ownership or registration receipt")
	}
	var runner string
	if err := q.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&runner); err != nil {
		return reserved, err
	}
	rows, err := q.Query("SELECT id,body FROM private_projects ORDER BY id")
	if err != nil {
		return reserved, err
	}
	projects := []storedPrivateProject{}
	for rows.Next() {
		var id string
		var raw []byte
		var p storedPrivateProject
		if err = rows.Scan(&id, &raw); err != nil {
			break
		}
		if json.Unmarshal(raw, &p) != nil {
			err = errors.New("invalid private-chain project body")
			break
		}
		canonical, _ := json.Marshal(p)
		if !bytes.Equal(raw, canonical) || p.Version != 1 || p.Binding.Validate() != nil || p.Binding.Mode != privatechain.Mode || p.Binding.RunnerID != runner || p.Binding.ProjectID != id || !strings.HasPrefix(id, "private-project-") || p.Binding.Generation != 1 || p.Revision != 1 || p.Phase != "planned" || p.CreatedAt.IsZero() || !protocol.ValidID(p.RegistrationOperationID) || len(p.Members) != len(p.Plan.Members) {
			err = errors.New("invalid private-chain project contract")
			break
		}
		if _, err = p.Plan.Digest(); err != nil {
			break
		}
		if p.Binding.FoundationDigest != p.Plan.FoundationDigest {
			err = errors.New("private-chain foundation binding differs")
			break
		}
		projects = append(projects, p)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return reserved, err
	}
	type claim struct {
		Project string
		Port    uint16
		Runtime string
	}
	expected := map[string]claim{}
	runtimeIDs := map[string]bool{}
	rows, err = q.Query("SELECT runtime_id FROM project_members")
	if err != nil {
		return reserved, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		runtimeIDs[id] = true
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return reserved, err
	}
	receipts := map[string]bool{}
	for _, p := range projects {
		var collisions int
		if err = q.QueryRow("SELECT count(*) FROM projects WHERE id=?", p.Binding.ProjectID).Scan(&collisions); err != nil {
			return reserved, err
		}
		if collisions != 0 || receipts[p.RegistrationOperationID] {
			return fail()
		}
		receipts[p.RegistrationOperationID] = true
		for i, m := range p.Members {
			_, decodeErr := hex.DecodeString(m.RuntimeID)
			if m.Member != p.Plan.Members[i] || !protocol.ValidPierName(m.InstanceID) || len(m.RuntimeID) != 32 || decodeErr != nil || strings.ToLower(m.RuntimeID) != m.RuntimeID || runtimeIDs[m.RuntimeID] || reserved.Instances[m.InstanceID] || reserved.HTTPPorts[m.HTTPPort] {
				return fail()
			}
			runtimeIDs[m.RuntimeID], reserved.Instances[m.InstanceID], reserved.HTTPPorts[m.HTTPPort] = true, true, true
			expected[m.InstanceID] = claim{p.Binding.ProjectID, m.HTTPPort, m.RuntimeID}
		}
		var actor, request, fingerprint string
		var raw, runtime []byte
		var op protocol.Operation
		if err = q.QueryRow("SELECT actor,request_id,fingerprint,body,runtime FROM operations WHERE id=?", p.RegistrationOperationID).Scan(&actor, &request, &fingerprint, &raw, &runtime); err != nil {
			return reserved, err
		}
		if json.Unmarshal(raw, &op) != nil || actor == "" || !protocol.ValidID(request) || len(runtime) != 0 {
			return fail()
		}
		want := protocol.Operation{ID: p.RegistrationOperationID, RequestID: request, ProjectID: p.Binding.ProjectID, Kind: privateProjectRegistrationKind, Actor: actor, Status: "succeeded", ConfigRevision: 1, CreatedAt: p.CreatedAt, FinishedAt: &p.CreatedAt}
		digest, _ := p.Plan.Digest()
		if !reflect.DeepEqual(op, want) || fingerprint != privateProjectFingerprint(privateProjectRegistration{RequestID: request, Plan: p.Plan, ReviewDigest: digest}) {
			return fail()
		}
	}
	rows, err = q.Query("SELECT instance_id,project_id,http_port,runtime_id FROM private_project_members ORDER BY instance_id")
	if err != nil {
		return reserved, err
	}
	for rows.Next() {
		var id string
		var actual claim
		if err = rows.Scan(&id, &actual.Project, &actual.Port, &actual.Runtime); err != nil {
			break
		}
		if want, ok := expected[id]; !ok || actual != want {
			err = errors.New("private-chain member differs from its reserved plan")
			break
		}
		delete(expected, id)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return reserved, err
	}
	if len(expected) != 0 {
		return fail()
	}
	rows, err = q.Query(`SELECT id FROM operations WHERE json_extract(body,'$.kind')=? OR fingerprint LIKE ?`, privateProjectRegistrationKind, `{"Kind":"`+privateProjectRegistrationKind+`",%`)
	if err != nil {
		return reserved, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		if !receipts[id] {
			err = errors.New("private-chain registration receipt lost its project")
			break
		}
		delete(receipts, id)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return reserved, err
	}
	if len(receipts) != 0 {
		return fail()
	}
	return reserved, nil
}

func (s *store) registerPrivateProject(actor string, req privateProjectRegistration) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(req.RequestID) {
		return protocol.Operation{}, false, bad("actor and valid request ID are required")
	}
	digest, err := req.Plan.Digest()
	if err != nil || req.ReviewDigest != digest {
		return protocol.Operation{}, false, bad("private-chain request must match its reviewed canonical plan digest")
	}
	fingerprint := privateProjectFingerprint(req)
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, e := replay(tx, actor, req.RequestID, fingerprint); found || e != nil {
		if e == nil {
			_, e = projectReservations(tx)
		}
		return op, found, e
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching private-chain project receipt exists for this actor")
	}
	reserved, err := projectReservations(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	all, err := instances(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, v := range all {
		reserved.Instances[v.Public.ID] = true
		reserved.HTTPPorts[v.Public.Config.HTTPPort] = true
		if v.Runtime.Prepared {
			reserved.HTTPPorts[v.Runtime.Port] = true
		}
		for _, c := range v.Runtime.Runtime.Containers {
			if !c.Removed {
				reserved.HTTPPorts[v.Runtime.Port] = true
			}
		}
	}
	request := privatechain.Request{Label: req.Plan.Label, Foundation: req.Plan.Foundation, ConfirmSharedAdministration: req.Plan.SharedAdministration}
	for _, m := range req.Plan.Members {
		request.Members = append(request.Members, m.MemberRequest)
	}
	if _, err = privatechain.Build(request, privatechain.Reservations{Instances: reserved.Instances, HTTPPorts: reserved.HTTPPorts}); err != nil {
		return protocol.Operation{}, false, conflict("project_conflict", err.Error())
	}
	for _, m := range req.Plan.Members {
		if !protocol.ValidPierName(m.InstanceID) {
			return protocol.Operation{}, false, bad("member instance names must start with a letter or number for their exclusive storage directory")
		}
		if err = s.availablePort("tcp", m.HTTPPort); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	p := storedPrivateProject{Version: 1, Binding: privatechain.Binding{Mode: privatechain.Mode, RunnerID: s.runnerID, ProjectID: "private-project-" + id, Generation: 1, FoundationDigest: req.Plan.FoundationDigest}, Revision: 1, Phase: "planned", Plan: req.Plan, RegistrationOperationID: "op-" + opID, CreatedAt: time.Now().UTC()}
	for _, m := range req.Plan.Members {
		runtimeID, e := randomID()
		if e != nil {
			return protocol.Operation{}, false, e
		}
		p.Members = append(p.Members, privateProjectMember{m, runtimeID})
	}
	body, err := json.Marshal(p)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("INSERT INTO private_projects(id,body) VALUES(?,?)", p.Binding.ProjectID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	for _, m := range p.Members {
		if _, err = tx.Exec("INSERT INTO private_project_members(instance_id,project_id,http_port,runtime_id) VALUES(?,?,?,?)", m.InstanceID, p.Binding.ProjectID, m.HTTPPort, m.RuntimeID); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	op := protocol.Operation{ID: p.RegistrationOperationID, RequestID: req.RequestID, ProjectID: p.Binding.ProjectID, Kind: privateProjectRegistrationKind, Actor: actor, Status: "succeeded", ConfigRevision: 1, CreatedAt: p.CreatedAt, FinishedAt: &p.CreatedAt}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = projectReservations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

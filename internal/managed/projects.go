package managed

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Project registration reserves the complete reviewed set without creating an
// instance, pier, bridge, anchor or container. Runtime dispatch is a later,
// revision-bound operation. The public registration route reconstructs this plan
// from primitive inputs; the original store fingerprint remains unchanged.
type projectRegistration struct {
	RequestID    string              `json:"request_id"`
	Plan         projectnetwork.Plan `json:"plan"`
	ReviewDigest string              `json:"review_digest"`
	ReplayOnly   bool                `json:"-"`
}

type projectMember struct {
	projectnetwork.Member
	RuntimeID string `json:"runtime_id"`
}

type storedProject struct {
	ID        string              `json:"id"`
	Revision  uint64              `json:"revision"`
	Phase     string              `json:"phase"`
	Plan      projectnetwork.Plan `json:"plan"`
	Members   []projectMember     `json:"members"`
	CreatedAt time.Time           `json:"created_at"`
}

func initializeProjects(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS project_members (
 instance_id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id),
 http_port INTEGER NOT NULL UNIQUE CHECK(http_port BETWEEN 1024 AND 65535),
 runtime_id TEXT NOT NULL UNIQUE);`)
	return err
}

// Read both ownership representations, refusing partial or altered records. An
// empty claims table must never make an existing project lose its reservations.
func fakeProjectReservations(q querier) (projectnetwork.Reservations, error) {
	reserved := projectnetwork.Reservations{Instances: map[string]bool{}, HTTPPorts: map[uint16]bool{}}
	type claim struct {
		Project string
		Port    uint16
		Runtime string
	}
	expected := map[string]claim{}
	runtimes := map[string]bool{}
	rows, err := q.Query("SELECT id,body FROM projects ORDER BY id")
	if err != nil {
		return reserved, err
	}
	for rows.Next() {
		var id string
		var data []byte
		var p storedProject
		if err = rows.Scan(&id, &data); err != nil {
			break
		}
		if json.Unmarshal(data, &p) != nil || p.ID != id || !protocol.ValidID(id) || p.Revision != 1 || p.Phase != "planned" || p.CreatedAt.IsZero() || len(p.Members) != len(p.Plan.Members) {
			err = errors.New("invalid stored project")
			break
		}
		if _, err = p.Plan.Digest(); err != nil {
			break
		}
		for i, m := range p.Members {
			_, decodeErr := hex.DecodeString(m.RuntimeID)
			if m.Member != p.Plan.Members[i] || len(m.RuntimeID) != 32 || decodeErr != nil || runtimes[m.RuntimeID] || reserved.Instances[m.InstanceID] || reserved.HTTPPorts[m.HTTPPort] {
				err = errors.New("invalid project member ownership")
				break
			}
			expected[m.InstanceID] = claim{id, m.HTTPPort, m.RuntimeID}
			runtimes[m.RuntimeID], reserved.Instances[m.InstanceID], reserved.HTTPPorts[m.HTTPPort] = true, true, true
		}
		if err != nil {
			break
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return reserved, err
	}
	rows, err = q.Query("SELECT instance_id,project_id,http_port,runtime_id FROM project_members ORDER BY instance_id")
	if err != nil {
		return reserved, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var actual claim
		if err = rows.Scan(&id, &actual.Project, &actual.Port, &actual.Runtime); err != nil {
			return reserved, err
		}
		if want, ok := expected[id]; !ok || actual != want {
			return reserved, errors.New("project membership differs from reserved plan")
		}
		delete(expected, id)
	}
	if err = rows.Err(); err != nil {
		return reserved, err
	}
	if len(expected) != 0 {
		return reserved, errors.New("project member reservation is missing")
	}
	return reserved, nil
}

func reserveProjectPort(q querier, port uint16) error {
	reserved, err := projectReservations(q)
	if err != nil {
		return err
	}
	if reserved.HTTPPorts[port] {
		return conflict("port_conflict", "HTTP port is reserved by a project member")
	}
	return nil
}

func (s *store) registerProject(actor string, req projectRegistration) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(req.RequestID) {
		return protocol.Operation{}, false, bad("actor and valid request ID are required")
	}
	digest, err := req.Plan.Digest()
	if err != nil || req.ReviewDigest != digest {
		return protocol.Operation{}, false, bad("project request must match the reviewed canonical plan digest")
	}
	data, _ := json.Marshal(struct {
		Kind    string
		Request projectRegistration
	}{"register-project", req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(data)); found || err != nil {
		return op, found, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching project receipt exists for this actor")
	}
	var duplicate int
	if err := tx.QueryRow("SELECT count(*) FROM projects WHERE lower(json_extract(body,'$.plan.label'))=lower(?)", req.Plan.Label).Scan(&duplicate); err != nil {
		return protocol.Operation{}, false, err
	}
	if duplicate > 0 {
		return protocol.Operation{}, false, conflict("project_conflict", "A project with this label already exists. Choose it or use another label.")
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
	request := projectnetwork.Request{Label: req.Plan.Label, ConfirmSharedAdministration: req.Plan.SharedAdministration}
	for _, m := range req.Plan.Members {
		request.Members = append(request.Members, m.MemberRequest)
	}
	if _, err = projectnetwork.Build(request, reserved); err != nil {
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
	p := storedProject{ID: "project-" + id, Revision: 1, Phase: "planned", Plan: req.Plan, CreatedAt: time.Now().UTC()}
	for _, m := range req.Plan.Members {
		runtimeID, err := randomID()
		if err != nil {
			return protocol.Operation{}, false, err
		}
		p.Members = append(p.Members, projectMember{Member: m, RuntimeID: runtimeID})
	}
	body, err := json.Marshal(p)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("INSERT INTO projects(id,body) VALUES(?,?)", p.ID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	for _, m := range p.Members {
		if _, err = tx.Exec("INSERT INTO project_members(instance_id,project_id,http_port,runtime_id) VALUES(?,?,?,?)", m.InstanceID, p.ID, m.HTTPPort, m.RuntimeID); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, ProjectID: p.ID, Kind: "register-project", Actor: actor, Status: "succeeded", ConfigRevision: 1, CreatedAt: p.CreatedAt, FinishedAt: &p.CreatedAt}
	if err = insertOperation(tx, op, string(data)); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

// Package service is the shared typed operation facade used by CLI and MCP.
// Runner admission, journals and policy remain authoritative in their existing
// packages. This facade never reads runner databases or shells out to the CLI.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/guidance"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectflow"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"github.com/google/jsonschema-go/jsonschema"
)

const MaxInputBytes = 128 << 10
const MaxResultBytes = 2 << 20

type Target struct {
	Name     string `json:"name"`
	RunnerID string `json:"runner_id"`
}

func (t Target) Validate() error {
	if !protocol.ValidID(t.Name) || !protocol.ValidID(t.RunnerID) {
		return errors.New("select an enrolled runner name and expected identity")
	}
	return nil
}

type Options struct {
	ConfigDir string
	// RequireAllowed makes an empty list deny all live targets. CLI calls select
	// their own explicit target; a persistent MCP server uses a startup allowlist.
	RequireAllowed  bool
	Allowed         []Target
	EnableMutations bool
	Guidance        *guidance.Bundle
	Sources         []*SourceRoot
}
type Descriptor struct {
	ID             string             `json:"id"`
	Description    string             `json:"description"`
	CLI            []string           `json:"cli"`
	MCP            string             `json:"mcp"`
	Effects        []string           `json:"effects"`
	Role           string             `json:"role"`
	Capabilities   []string           `json:"capabilities"`
	CapabilityRule string             `json:"capability_rule,omitempty"`
	Bindings       string             `json:"bindings"`
	Replay         string             `json:"replay"`
	InputSchema    *jsonschema.Schema `json:"input_schema"`
	OutputSchema   *jsonschema.Schema `json:"output_schema"`
}
type Problem struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Dispatch     string `json:"dispatch"`
	OriginalCode string `json:"original_code,omitempty"`
	NextAction   string `json:"next_action"`
}
type Result struct {
	Version   int      `json:"version"`
	Operation string   `json:"operation"`
	Data      any      `json:"data,omitempty"`
	Error     *Problem `json:"error,omitempty"`
}
type envelope[T any] struct {
	Version   int      `json:"version"`
	Operation string   `json:"operation"`
	Data      *T       `json:"data,omitempty"`
	Error     *Problem `json:"error,omitempty"`
}
type operation struct {
	descriptor Descriptor
	invoke     func(context.Context, *Service, json.RawMessage) (any, error)
	mutates    bool
	output     *jsonschema.Resolved
}
type Service struct {
	options    Options
	operations map[string]operation
}

// New performs no I/O. Offline catalog discovery does not create registry state.
func New(options Options) (*Service, error) {
	seen := map[Target]bool{}
	for _, t := range options.Allowed {
		if t.Validate() != nil || seen[t] {
			return nil, errors.New("invalid or duplicate allowed runner")
		}
		seen[t] = true
	}
	options.Allowed = slices.Clone(options.Allowed)
	options.Sources = slices.Clone(options.Sources)
	seenSources := map[string]bool{}
	for _, source := range options.Sources {
		if source == nil || source.root == nil || !protocol.ValidID(source.Name) || seenSources[source.Name] {
			return nil, errors.New("invalid or duplicate approved source root")
		}
		seenSources[source.Name] = true
	}
	for i, source := range options.Sources {
		frozen := *source
		options.Sources[i] = &frozen
	}
	s := &Service{options: options, operations: map[string]operation{}}
	if err := s.register(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Service) Catalog() []Descriptor {
	out := make([]Descriptor, 0, len(s.operations))
	for _, op := range s.operations {
		out = append(out, op.descriptor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (s *Service) MutationsEnabled() bool { return s.options.EnableMutations }
func (s *Service) Call(ctx context.Context, id string, input json.RawMessage) Result {
	out := Result{Version: 1, Operation: id}
	op, ok := s.operations[id]
	if !ok {
		out.Error = &Problem{"unknown_operation", "Unknown core operation.", "not-dispatched", "", "Read the operation catalog."}
		return out
	}
	if op.mutates && !s.options.EnableMutations {
		out.Error = &Problem{"mutation_disabled", "This server has not enabled mutation tools.", "not-dispatched", "", "Enable only the required operation authority at server startup."}
		return out
	}
	if len(input) == 0 {
		input = json.RawMessage(`{}`)
	}
	if len(input) > MaxInputBytes {
		out.Error = &Problem{"invalid_input", "Input exceeds the bounded operation limit.", "not-dispatched", "", "Use the operation's typed schema."}
		return out
	}
	data, err := op.invoke(ctx, s, input)
	if err != nil {
		out.Error = problem(err, op.mutates)
		return out
	}
	out.Data = data
	encoded, err := json.Marshal(out)
	if err != nil || len(encoded) > MaxResultBytes {
		out.Data = nil
		out.Error = &Problem{"result_too_large", "The bounded result could not be returned.", "completed-read-or-unknown-mutation", "", "Inspect the saved receipt with the original IDs; do not repeat a mutation."}
		return out
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		out.Data = nil
		out.Error = &Problem{"invalid_output", "The result could not be verified.", "response-received", "", "Inspect saved status with the original IDs before further effects."}
		return out
	}
	value, err = schemaNumbers(value)
	if err != nil || op.output.Validate(value) != nil {
		out.Data = nil
		out.Error = &Problem{"invalid_output", "The result did not match its typed contract.", "response-received", "", "Inspect saved status with the original IDs before further effects."}
	}
	return out
}

type refusal struct{ code, message, next string }

type beforeEffect struct{ err error }

func (e *beforeEffect) Error() string { return e.err.Error() }
func (e *beforeEffect) Unwrap() error { return e.err }

func (e *refusal) Error() string { return e.message }
func invalid() error {
	return &refusal{"invalid_input", "Input does not match the typed operation contract.", "Read the schema and preserve the reviewed IDs and bindings."}
}
func problem(err error, mutates bool) *Problem {
	if errors.Is(err, projectflow.ErrHistory) {
		return &Problem{"creation_history_changed", "Saved creation history or bindings changed.", "not-dispatched", "", "Read project.status.v1 and its supported recovery source/checkpoint; no new phase was submitted."}
	}
	var staging *collection.DeskStageAdmissionError
	if errors.As(err, &staging) {
		out := problem(staging.Err, false)
		out.Dispatch = "not-dispatched"
		return out
	}
	var before *beforeEffect
	if errors.As(err, &before) {
		out := problem(before.err, false)
		out.Dispatch = "not-dispatched"
		return out
	}
	var no *refusal
	if errors.As(err, &no) {
		return &Problem{no.code, no.message, "not-dispatched", "", no.next}
	}
	var uncertain *client.OutcomeUncertainError
	if errors.As(err, &uncertain) {
		return &Problem{"outcome_uncertain", "The response did not establish the operation outcome.", "outcome-uncertain", "", "Read saved status or perform exact-request lookup using the original request; never mint a replacement ID."}
	}
	if errors.Is(err, client.ErrIdentityMismatch) {
		dispatch := "read-failed"
		if mutates {
			dispatch = "outcome-uncertain"
		}
		return &Problem{"identity_mismatch", "Runner identity or mode differs from enrollment.", dispatch, "", "Review the selected enrollment and original receipt without re-enrolling automatically."}
	}
	var unsupported *client.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		return &Problem{"capability_unavailable", "The selected runner lacks a required capability.", "not-dispatched", "", "Read Hello and application preflight before preparing effects."}
	}
	var remote *client.RemoteError
	if errors.As(err, &remote) {
		code := remote.Problem.Code
		if !protocol.ValidID(code) {
			code = "runner_error"
		}
		return &Problem{"runner_rejected", "The runner rejected this request.", "runner-reported-error", code, "Read the original receipt and current policy/revision before deciding how to proceed."}
	}
	dispatch := "read-failed"
	next := "Repeat the supported read after checking the selected connection."
	if mutates {
		dispatch = "outcome-uncertain"
		next = "Inspect the saved receipt with the original IDs before any further mutation."
	}
	// Do not reflect arbitrary transport/runner error strings into model-visible
	// transcripts: those can contain paths, inputs or authentication material.
	return &Problem{"operation_failed", "The operation did not produce a verified result.", dispatch, "", next}
}
func (s *Service) registry() (*collection.Registry, error) {
	if s.options.ConfigDir == "" {
		return nil, &refusal{"registry_required", "This operation requires a configured registry.", "Set --config-dir to the intended private registry."}
	}
	if info, err := os.Lstat(s.options.ConfigDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, &refusal{"registry_unavailable", "The selected registry directory is unavailable.", "Use the existing enrolled private registry; discovery never enrolls or creates one."}
	}
	return collection.Open(s.options.ConfigDir)
}
func (s *Service) selected(ctx context.Context, t Target) (*collection.Registry, collection.Connection, error) {
	if t.Validate() != nil {
		return nil, collection.Connection{}, invalid()
	}
	if s.options.RequireAllowed && !slices.Contains(s.options.Allowed, t) {
		return nil, collection.Connection{}, &refusal{"target_not_allowed", "The selected runner is outside this server's allowlist.", "Use an explicitly authorized runner name and identity."}
	}
	r, err := s.registry()
	if err != nil {
		return nil, collection.Connection{}, err
	}
	e, err := r.Get(ctx, t.Name)
	if err != nil {
		return nil, e, err
	}
	if e.RunnerID != t.RunnerID {
		return nil, e, &refusal{"identity_mismatch", "Selected enrollment no longer has the expected runner identity.", "Review the selected enrollment without re-enrolling automatically."}
	}
	return r, e, nil
}
func (s *Service) authorize(ctx context.Context, t Target, d Descriptor) (*collection.Registry, collection.Connection, error) {
	r, e, err := s.selected(ctx, t)
	if err != nil {
		return nil, e, err
	}
	c := e.Client(true)
	defer c.Close()
	h, err := c.Hello(ctx)
	if err != nil {
		return nil, e, err
	}
	if err = e.Identity().Check(h.RunnerID, h.Mode); err != nil {
		return nil, e, err
	}
	if d.Role == "operator" && h.AccessRole != "operator" {
		return nil, e, &refusal{"operator_required", "The selected key is not an operator.", "Use Setup to pair an appropriate restricted operator key."}
	}
	for _, feature := range d.Capabilities {
		if !slices.Contains(h.Features, feature) {
			return nil, e, &client.UnsupportedCapabilityError{Feature: feature, Hello: h}
		}
	}
	return r, e, nil
}
func bind[I, O any](s *Service, d Descriptor, handler func(context.Context, *Service, I) (O, error)) error {
	if _, exists := s.operations[d.ID]; exists {
		return errors.New("duplicate core operation")
	}
	in, err := jsonschema.For[I](nil)
	if err != nil {
		return err
	}
	out, err := jsonschema.For[envelope[O]](nil)
	if err != nil {
		return err
	}
	resolved, err := in.Resolve(nil)
	if err != nil {
		return err
	}
	output, err := out.Resolve(nil)
	if err != nil {
		return err
	}
	d.InputSchema, d.OutputSchema = in, out
	d.CLI = append([]string{"omabit call " + d.ID}, d.CLI...)
	d.MCP = "omabit." + d.ID
	mutates := slices.Contains(d.Effects, "journal-write") || slices.Contains(d.Effects, "runtime-mutation") || slices.Contains(d.Effects, "local-cache-write") || slices.Contains(d.Effects, "policy-edit")
	s.operations[d.ID] = operation{descriptor: d, mutates: mutates, output: output, invoke: func(ctx context.Context, s *Service, raw json.RawMessage) (any, error) {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF {
			return nil, invalid()
		}
		value, err := schemaNumbers(value)
		if err != nil || resolved.Validate(value) != nil {
			return nil, invalid()
		}
		var input I
		decoder = json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
			return nil, invalid()
		}
		if validator, ok := any(input).(interface{ Validate() error }); ok && validator.Validate() != nil {
			return nil, invalid()
		}
		return handler(ctx, s, input)
	}}
	return nil
}

// jsonschema-go v0.4.3 recognizes native integers for type validation, while
// json.Number is treated as a string. Keep integers exact rather than decoding
// every number through float64 and rounding revision/inode bindings above2^53.
func schemaNumbers(value any) (any, error) {
	switch v := value.(type) {
	case json.Number:
		if integer, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return integer, nil
		}
		if integer, err := strconv.ParseUint(string(v), 10, 64); err == nil {
			return integer, nil
		}
		return v.Float64()
	case map[string]any:
		for key, item := range v {
			next, err := schemaNumbers(item)
			if err != nil {
				return nil, err
			}
			v[key] = next
		}
		return v, nil
	case []any:
		for i, item := range v {
			next, err := schemaNumbers(item)
			if err != nil {
				return nil, err
			}
			v[i] = next
		}
		return v, nil
	default:
		return value, nil
	}
}

type RunnerSummary struct {
	Target
	Mode       string    `json:"mode"`
	Revision   uint64    `json:"revision"`
	EnrolledAt time.Time `json:"enrolled_at"`
}

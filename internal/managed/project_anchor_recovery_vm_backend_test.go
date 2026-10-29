//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Only installed on the already admitted, no-NIC disposable VM backend.
// Every mutation checks the actual SQLite checkpoint before calling the adapter.
type vmAnchorRecoveryBackend struct {
	publicCLI                         *vmPublicRecoveryCLI
	store                             *store
	project                           string
	register                          func(projectnetwork.RecoveredMember, string)
	events                            []vmResumedReadinessEvent
	loseSignal, loseCreate, loseStart bool
}

func (b *vmAnchorRecoveryBackend) record(action string, start time.Time, err error) {
	e := vmResumedReadinessEvent{Action: action, StartedAt: start, FinishedAt: time.Now().UTC()}
	if err != nil {
		e.Error = vmLaunchError(err)
	}
	b.events = append(b.events, e)
}
func (b *vmAnchorRecoveryBackend) check(field string, state any) error {
	jobs, err := projectAnchorRecoveryJobs(b.store.db)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.ProjectID != b.project {
			continue
		}
		var retained any
		switch field {
		case "drain":
			if j.Journal.Drain != nil {
				retained = *j.Journal.Drain
			}
		case "namespace":
			if j.Journal.Namespace != nil {
				retained = *j.Journal.Namespace
			}
		case "retirement":
			if j.Journal.Retirement != nil {
				retained = *j.Journal.Retirement
			}
		case "replacement":
			if j.Journal.Replacement != nil {
				retained = *j.Journal.Replacement
			}
		}
		if reflect.DeepEqual(retained, state) {
			// Full history is audited before/after the experiment and at reopen.
			// Do not add a redundant audit inside the fresh engine observation bracket.
			return nil
		}
	}
	return errors.New("recovery effect preceded exact committed checkpoint")
}

func (b *vmAnchorRecoveryBackend) OpenProjectAnchorRecoveryReview(ctx context.Context, s runtimequalify.ProjectAnchorRecoveryReviewSelection) (out projectnetwork.AnchorRecoveryObserver, err error) {
	start := time.Now()
	defer func() { b.record("open-review", start, err) }()
	d, err := runtimequalify.OpenProjectAnchorRecoveryReview(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryReviewDriver{backend: b, driver: d}, nil
}

type vmRecoveryReviewDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorRecoveryObserver
}

func (d *vmRecoveryReviewDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now()
	defer func() { d.backend.record("review-VerifyEngine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryReviewDriver) InspectAnchorRecovery(ctx context.Context, s projectnetwork.AnchorRecoveryContext) (value projectnetwork.AnchorRecoveryObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("review-InspectAnchorRecovery", start, err) }()
	return d.driver.InspectAnchorRecovery(ctx, s)
}

func (b *vmAnchorRecoveryBackend) OpenProjectAnchorDrain(ctx context.Context, s runtimequalify.ProjectAnchorDrainSelection) (out projectnetwork.AnchorDrainDriver, err error) {
	start := time.Now()
	defer func() { b.record("open-drain", start, err) }()
	d, err := runtimequalify.OpenProjectAnchorDrain(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryDrainDriver{backend: b, driver: d}, nil
}

type vmRecoveryDrainDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorDrainDriver
}

func (d *vmRecoveryDrainDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now()
	defer func() { d.backend.record("drain-VerifyEngine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryDrainDriver) InspectDrain(ctx context.Context, s projectnetwork.AnchorDrain) (value projectnetwork.AnchorDrainObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("drain-InspectDrain", start, err) }()
	return d.driver.InspectDrain(ctx, s)
}
func (d *vmRecoveryDrainDriver) SignalDrainMember(ctx context.Context, s projectnetwork.AnchorDrain, i int) (err error) {
	start := time.Now()
	defer func() { d.backend.record("drain-SignalDrainMember", start, err) }()
	if err = d.backend.check("drain", s); err != nil {
		return
	}
	err = d.driver.SignalDrainMember(ctx, s, i)
	if err == nil && d.backend.loseSignal {
		d.backend.loseSignal = false
		return errors.New("fixture lost successful signal reply")
	}
	return err
}
func (d *vmRecoveryDrainDriver) AwaitDrainExit(ctx context.Context, s projectnetwork.AnchorDrain, i int) (value projectnetwork.AnchorDrainObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("drain-AwaitDrainExit", start, err) }()
	return d.driver.AwaitDrainExit(ctx, s, i)
}

func (b *vmAnchorRecoveryBackend) OpenProjectAnchorNamespace(ctx context.Context, s runtimequalify.ProjectAnchorNamespaceSelection) (out projectnetwork.AnchorNamespaceDriver, err error) {
	start := time.Now()
	defer func() { b.record("open-namespace", start, err) }()
	d, err := runtimequalify.OpenProjectAnchorNamespace(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryNamespaceDriver{backend: b, driver: d}, nil
}

type vmRecoveryNamespaceDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorNamespaceDriver
}

func (d *vmRecoveryNamespaceDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now()
	defer func() { d.backend.record("namespace-VerifyEngine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryNamespaceDriver) InspectNamespace(ctx context.Context, s projectnetwork.AnchorNamespace) (value projectnetwork.AnchorDrainObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("namespace-InspectNamespace", start, err) }()
	return d.driver.InspectNamespace(ctx, s)
}
func (d *vmRecoveryNamespaceDriver) StartRecoveryAnchor(ctx context.Context, s projectnetwork.AnchorNamespace) (err error) {
	start := time.Now()
	defer func() { d.backend.record("namespace-StartRecoveryAnchor", start, err) }()
	if err = d.backend.check("namespace", s); err != nil {
		return
	}
	return d.driver.StartRecoveryAnchor(ctx, s)
}

func (b *vmAnchorRecoveryBackend) OpenProjectAnchorRetirement(ctx context.Context, s runtimequalify.ProjectAnchorRetirementSelection) (out projectnetwork.AnchorRetirementDriver, err error) {
	start := time.Now()
	defer func() { b.record("open-retirement", start, err) }()
	d, err := runtimequalify.OpenProjectAnchorRetirement(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryRetirementDriver{backend: b, driver: d}, nil
}

type vmRecoveryRetirementDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorRetirementDriver
}

func (d *vmRecoveryRetirementDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now()
	defer func() { d.backend.record("retirement-VerifyEngine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryRetirementDriver) InspectRetirement(ctx context.Context, s projectnetwork.AnchorRetirement) (value projectnetwork.AnchorRetirementObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("retirement-InspectRetirement", start, err) }()
	return d.driver.InspectRetirement(ctx, s)
}
func (d *vmRecoveryRetirementDriver) RemoveRetiredMember(ctx context.Context, s projectnetwork.AnchorRetirement, i int) (err error) {
	start := time.Now()
	defer func() { d.backend.record("retirement-RemoveRetiredMember", start, err) }()
	if err = d.backend.check("retirement", s); err != nil {
		return
	}
	return d.driver.RemoveRetiredMember(ctx, s, i)
}

func (b *vmAnchorRecoveryBackend) OpenProjectAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementSelection) (out projectnetwork.AnchorReplacementDriver, err error) {
	start := time.Now()
	defer func() { b.record("open-replacement", start, err) }()
	d, err := runtimequalify.OpenProjectAnchorReplacement(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryReplacementDriver{backend: b, driver: d}, nil
}

type vmRecoveryReplacementDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorReplacementDriver
}

func (d *vmRecoveryReplacementDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now()
	defer func() { d.backend.record("replacement-VerifyEngine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryReplacementDriver) InspectReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int, stage string) (value projectnetwork.AnchorRetirementObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("replacement-InspectReplacement", start, err) }()
	return d.driver.InspectReplacement(ctx, s, i, stage)
}
func (d *vmRecoveryReplacementDriver) CreateReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int) (value string, err error) {
	start := time.Now()
	defer func() { d.backend.record("replacement-CreateReplacement", start, err) }()
	if err = d.backend.check("replacement", s); err != nil {
		return
	}
	value, err = d.driver.CreateReplacement(ctx, s, i)
	if value != "" {
		d.backend.register(s.Members[i], value)
		if err == nil && d.backend.loseCreate {
			d.backend.loseCreate = false
			err = errors.New("fixture Create reply included ID and error")
		}
	}
	return
}
func (d *vmRecoveryReplacementDriver) StartReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int) (err error) {
	start := time.Now()
	defer func() { d.backend.record("replacement-StartReplacement", start, err) }()
	if err = d.backend.check("replacement", s); err != nil {
		return
	}
	return d.driver.StartReplacement(ctx, s, i)
}

func (b *vmAnchorRecoveryBackend) OpenProjectAnchorRecoveryReadiness(ctx context.Context, s runtimequalify.ProjectAnchorRecoveryReadinessSelection) (out projectnetwork.AnchorRecoveryReadinessDriver, err error) {
	start := time.Now()
	defer func() { b.record("open-readiness", start, err) }()
	d, err := runtimequalify.OpenProjectAnchorRecoveryReadiness(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryReadinessDriver{backend: b, driver: d}, nil
}

type vmRecoveryReadinessDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorRecoveryReadinessDriver
}

func (d *vmRecoveryReadinessDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now()
	defer func() { d.backend.record("readiness-VerifyEngine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryReadinessDriver) InspectRecoveryReadiness(ctx context.Context, s projectnetwork.AnchorReplacement) (value projectnetwork.AnchorRetirementObservation, err error) {
	start := time.Now()
	defer func() { d.backend.record("readiness-InspectRecoveryReadiness", start, err) }()
	return d.driver.InspectRecoveryReadiness(ctx, s)
}
func (d *vmRecoveryReadinessDriver) ReadHostIdentity(ctx context.Context, instance string) (value string, err error) {
	start := time.Now()
	defer func() { d.backend.record("readiness-ReadHostIdentity", start, err) }()
	return d.driver.ReadHostIdentity(ctx, instance)
}

func (b *vmAnchorRecoveryBackend) PlanProjectAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementPlanSelection) (out projectnetwork.AnchorReplacement, err error) {
	start := time.Now()
	defer func() { b.record("plan-replacement", start, err) }()
	return runtimequalify.PlanProjectAnchorReplacement(ctx, s)
}

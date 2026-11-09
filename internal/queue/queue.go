package queue

import (
	"context"
	"dl-manager/internal/model"
	"dl-manager/internal/storage"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runner interface {
	Run(context.Context, model.Job, func(model.Event)) error
}
type activeJob struct {
	cancel   context.CancelFunc
	resource string
	profile  string
	target   string
}
type Manager struct {
	mu        sync.Mutex
	store     *storage.Store
	runner    Runner
	active    map[string]*activeJob
	running   bool
	closed    bool
	resetting bool
	lastError string
	wake      chan struct{}
	done      chan struct{}
	wg        sync.WaitGroup
}

func New(s *storage.Store, r Runner) *Manager {
	m := &Manager{store: s, runner: r, active: map[string]*activeJob{}, wake: make(chan struct{}, 1), done: make(chan struct{})}
	go m.loop()
	return m
}
func (m *Manager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}
func (m *Manager) loop() {
	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	defer close(m.done)
	for {
		select {
		case <-m.wake:
		case <-tick.C:
		}
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			return
		}
		m.dispatch()
		m.mu.Unlock()
	}
}
func resource(j model.Job) string {
	return strings.ToLower(filepath.Clean(j.Destination)) + "|" + j.Site
}
func (m *Manager) fail(e error) { m.lastError = e.Error(); m.stopLocked("interrupted") }
func (m *Manager) dispatch() {
	if !m.running {
		return
	}
	state, e := m.store.Snapshot()
	if e != nil {
		m.fail(e)
		return
	}
	if state.Settings.QuotaBytes > 0 && state.UsedBytes >= state.Settings.QuotaBytes {
		m.stopLocked("paused_quota")
		return
	}
	for _, j := range state.Jobs {
		if len(m.active) >= state.Settings.Concurrency {
			return
		}
		if j.Status != "queued" {
			continue
		}
		if _, ok := m.active[j.ID]; ok {
			continue
		}
		j.Destination = j.EffectiveDestination(state.Settings)
		if j.Destination == "" || !filepath.IsAbs(j.Destination) {
			j.Status = "failed"
			j.Error = "Choisissez un dossier global ou un dossier propre au site"
			if e = m.store.PutJob(j); e != nil {
				m.fail(e)
				return
			}
			continue
		}
		j.ProfileMode = j.ProfileSelection()
		profile, err := model.ResolveProfile(j, state)
		if err != nil {
			j.Status = "authentication_required"
			j.Error = err.Error()
			if e = m.store.PutJob(j); e != nil {
				m.fail(e)
				return
			}
			continue
		}
		j.ProfileID = profile
		busy := false
		key := resource(j)
		for _, a := range m.active {
			if a.resource == key || (j.ProfileID != "" && a.profile == j.ProfileID) {
				busy = true
				break
			}
		}
		if busy {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		a := &activeJob{cancel: cancel, resource: key, profile: j.ProfileID}
		j.Status = "running"
		j.Error = ""
		j.Started = model.Now()
		j.Ended = ""
		if e = m.store.PutJob(j); e != nil {
			cancel()
			m.fail(e)
			return
		}
		m.active[j.ID] = a
		m.wg.Add(1)
		go m.run(ctx, j)
	}
}
func (m *Manager) run(ctx context.Context, j model.Job) {
	defer m.wg.Done()
	err := m.runner.Run(ctx, j, func(e model.Event) { m.event(j.ID, e) })
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.active[j.ID]
	if a == nil {
		return
	}
	state, e := m.store.Snapshot()
	if e != nil {
		delete(m.active, j.ID)
		m.fail(e)
		return
	}
	for _, v := range state.Jobs {
		if v.ID == j.ID {
			j = v
			break
		}
	}
	if a.target != "" {
		j.Status = a.target
	} else if err != nil {
		j.Status = "failed"
		var classified interface{ Status() string }
		if errors.As(err, &classified) {
			j.Status = classified.Status()
		}
		j.Error = err.Error()
	} else {
		j.Status = "completed"
		j.Error = ""
	}
	j.Ended = model.Now()
	j.Current = ""
	if e = m.store.PutJob(j); e != nil {
		m.fail(e)
	}
	delete(m.active, j.ID)
	if j.Status == "authentication_required" && j.ProfileID != "" {
		for _, p := range state.Profiles {
			if p.ID == j.ProfileID {
				p.State = "authentication_required"
				if e = m.store.PutProfile(p); e != nil {
					m.fail(e)
				}
			}
		}
	}
	m.signal()
}
func (m *Manager) event(id string, ev model.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.active[id]; !ok {
		return
	}
	state, e := m.store.Snapshot()
	if e != nil {
		m.fail(e)
		return
	}
	var j model.Job
	found := false
	for _, v := range state.Jobs {
		if v.ID == id {
			j = v
			found = true
			break
		}
	}
	if !found {
		return
	}
	switch ev.Kind {
	case "bytes":
		if ev.Bytes <= 0 {
			return
		}
		j.Bytes += ev.Bytes
		if e = m.store.Progress(j, ev.Bytes); e != nil {
			m.fail(e)
			return
		}
		state.UsedBytes += ev.Bytes
	case "file":
		j.Files++
	case "skip":
		j.Skipped++
	case "current":
		j.Current = ev.Message
	case "exit":
		j.ExitCode = ev.ExitCode
	case "log":
		if e = m.store.Log(id, ev.Message); e != nil {
			m.fail(e)
		}
		return
	}
	if e = m.store.PutJob(j); e != nil {
		m.fail(e)
		return
	}
	if state.Settings.QuotaBytes > 0 && state.UsedBytes >= state.Settings.QuotaBytes {
		m.stopLocked("paused_quota")
	}
}
func (m *Manager) stopLocked(status string) {
	m.running = false
	for _, a := range m.active {
		if a.target == "" {
			a.target = status
		}
		a.cancel()
	}
}
func (m *Manager) Snapshot() (model.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, e := m.store.Snapshot()
	for i := range s.Jobs {
		s.Jobs[i].Destination = s.Jobs[i].EffectiveDestination(s.Settings)
	}
	s.Running = m.running
	s.Active = len(m.active)
	s.LastError = m.lastError
	return s, e
}
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, active := range m.active {
		if active.target != "" {
			return errors.New("Arrêt des tâches en cours : attendez leur mise en pause avant de reprendre")
		}
	}
	if m.closed || m.resetting {
		return errors.New("Arrêt ou remise à zéro en cours")
	}
	s, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	if s.Settings.QuotaBytes > 0 && s.UsedBytes >= s.Settings.QuotaBytes {
		return errors.New("Limite atteinte : réinitialisez le compteur ou augmentez le quota")
	}
	for _, j := range s.Jobs {
		if j.Status == "paused" || j.Status == "interrupted" || j.Status == "paused_quota" || j.Status == "authentication_required" {
			j.Status = "queued"
			if e = m.store.PutJob(j); e != nil {
				return e
			}
		}
	}
	m.lastError = ""
	m.running = true
	m.signal()
	return nil
}
func (m *Manager) Pause() { m.mu.Lock(); m.stopLocked("paused"); m.mu.Unlock() }
func (m *Manager) ResetBudget() error {
	m.mu.Lock()
	if m.resetting || m.closed {
		m.mu.Unlock()
		return errors.New("Arrêt ou remise à zéro en cours")
	}
	m.resetting = true
	m.stopLocked("paused")
	m.mu.Unlock()
	m.wg.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()
	defer func() { m.resetting = false }()
	return m.store.ResetBudget()
}
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	m.stopLocked("interrupted")
	m.mu.Unlock()
	m.signal()
	<-m.done
	m.wg.Wait()
}
func (m *Manager) Add(j model.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, e := model.ParseURL(j.URL); e != nil {
		return e
	}
	if e := j.Filters.Validate(); e != nil {
		return e
	}
	if len(j.Filters.Allowed()) == 0 {
		return errors.New("Aucune extension sélectionnée pour les médias activés")
	}
	if j.Destination == "" || !filepath.IsAbs(j.Destination) {
		return errors.New("Choisissez un dossier de destination absolu")
	}
	if !model.ValidLayout(j.Layout) || !model.ValidFilename(j.Filename) {
		return errors.New("Nommage invalide")
	}
	s, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	for _, old := range s.Jobs {
		if old.URL == j.URL && !model.Terminal(old.Status) {
			return errors.New("Cette adresse est déjà dans la file")
		}
		if old.Order >= j.Order {
			j.Order = old.Order + 1
		}
	}
	j.ProfileMode = j.ProfileSelection()
	if _, e := model.ResolveProfile(j, s); e != nil {
		return e
	}
	if j.ID == "" {
		j.ID = model.ID()
	}
	j.Status = "queued"
	j.Added = model.Now()
	if e = m.store.PutJob(j); e != nil {
		return e
	}
	m.signal()
	return nil
}
func (m *Manager) Action(id, action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	idx := -1
	for i, j := range s.Jobs {
		if j.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("Tâche introuvable")
	}
	j := s.Jobs[idx]
	a, active := m.active[id]
	if active {
		if action == "pause" || action == "cancel" {
			a.target = "paused"
			if action == "cancel" {
				a.target = "cancelled"
			}
			a.cancel()
			return nil
		}
		return errors.New("Mettez la tâche en pause avant cette action")
	}
	switch action {
	case "pause":
		j.Status = "paused"
	case "cancel":
		j.Status = "cancelled"
	case "retry":
		j.Status = "queued"
		j.Error = ""
	case "delete":
		return m.store.DeleteJob(id)
	case "up", "down":
		other := idx - 1
		if action == "down" {
			other = idx + 1
		}
		if other < 0 || other >= len(s.Jobs) {
			return nil
		}
		peer := s.Jobs[other]
		j.Order, peer.Order = peer.Order, j.Order
		if e = m.store.PutJob(peer); e != nil {
			return e
		}
	default:
		return fmt.Errorf("Action inconnue : %s", action)
	}
	e = m.store.PutJob(j)
	m.signal()
	return e
}
func (m *Manager) SaveSettings(s model.Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.store.SaveSettings(s); e != nil {
		return e
	}
	state, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	if s.QuotaBytes > 0 && state.UsedBytes >= s.QuotaBytes {
		m.stopLocked("paused_quota")
	}
	excess := len(m.active) - s.Concurrency
	for _, a := range m.active {
		if excess <= 0 {
			break
		}
		a.target = "paused"
		a.cancel()
		excess--
	}
	m.signal()
	return nil
}
func (m *Manager) ProfileBusy(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.active {
		if a.profile == id {
			return true
		}
	}
	return false
}

// WithProfileMutation serializes credential replacement/deletion with dispatch.
// The callback must use the Store/Vault directly, not re-enter the Manager.
func (m *Manager) WithProfileMutation(id string, change func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("Application en cours de fermeture")
	}
	for _, active := range m.active {
		if active.profile == id {
			return errors.New("Mettez en pause les tâches utilisant ce compte")
		}
	}
	return change()
}

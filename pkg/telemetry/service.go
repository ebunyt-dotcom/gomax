// Package telemetry implements the optional Max telemetry stream used by
// PyMax. It intentionally runs in the background and never turns a telemetry
// failure into a client failure.
package telemetry

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/ebunyt-dotcom/gomax/pkg/api"
	"github.com/ebunyt-dotcom/gomax/pkg/protocol"
	"github.com/ebunyt-dotcom/gomax/pkg/types"
)

// Screen is the numeric screen identifier expected by Max telemetry.
type Screen int

const (
	ScreenBackground Screen = 1
	ScreenContacts   Screen = 100
	ScreenChats      Screen = 150
	ScreenSearch     Screen = 151
	ScreenCalls      Screen = 300
	ScreenChat       Screen = 350
	ScreenSettings   Screen = 450
	ScreenMiniApp    Screen = 500
)

// Event is one LOG payload event.
type Event struct {
	Time      int64          `json:"time" msgpack:"time"`
	UserID    int64          `json:"userId" msgpack:"userId"`
	Type      string         `json:"type" msgpack:"type"`
	Event     string         `json:"event" msgpack:"event"`
	Params    map[string]any `json:"params" msgpack:"params"`
	SessionID int            `json:"sessionId" msgpack:"sessionId"`
}

// Snapshot is the authenticated state needed to render telemetry events.
type Snapshot struct {
	Ready  bool
	UserID int64
	Chats  []types.Chat
}

// Timing mirrors PyMax's internal timing and is exported to make deterministic
// tests possible. Zero fields are filled from DefaultTiming.
type Timing struct {
	StartupMin        time.Duration
	StartupMax        time.Duration
	SessionIdleMin    time.Duration
	SessionIdleMax    time.Duration
	RenderMin         time.Duration
	RenderMax         time.Duration
	ReturnMin         time.Duration
	ReturnMax         time.Duration
	ReturnChance      float64
	ChatsRenderChance float64
}

var DefaultTiming = Timing{
	StartupMin: 15 * time.Second, StartupMax: 90 * time.Second,
	SessionIdleMin: 15 * time.Minute, SessionIdleMax: 45 * time.Minute,
	RenderMin: 150 * time.Millisecond, RenderMax: 1200 * time.Millisecond,
	ReturnMin: 12 * time.Second, ReturnMax: 45 * time.Second,
	ReturnChance: 0.40, ChatsRenderChance: 0.20,
}

// Service sends randomized, mobile-shaped navigation telemetry while the
// owning client remains authenticated.
type Service struct {
	invoker  api.Invoker
	snapshot func() Snapshot
	timing   Timing

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewService(invoker api.Invoker, snapshot func() Snapshot) *Service {
	return NewServiceWithTiming(invoker, snapshot, DefaultTiming)
}

func NewServiceWithTiming(invoker api.Invoker, snapshot func() Snapshot, timing Timing) *Service {
	return &Service{invoker: invoker, snapshot: snapshot, timing: normalizeTiming(timing)}
}

// Start starts one telemetry loop. Repeated calls while running are ignored.
func (s *Service) Start(parent context.Context, sessionID int) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	s.cancel, s.done = cancel, done
	s.mu.Unlock()
	go s.run(ctx, done, sessionID)
}

// Stop cancels and waits for the telemetry goroutine.
func (s *Service) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func (s *Service) run(ctx context.Context, done chan struct{}, sessionID int) {
	defer s.finish(done)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	if !sleepBetween(ctx, rng, s.timing.StartupMin, s.timing.StartupMax) {
		return
	}
	state := s.snapshot()
	if !state.Ready || state.UserID == 0 {
		return
	}
	s.send(ctx, []Event{LoginEvent(state.UserID, sessionID)})

	planner := newPlanner()
	actionID := 0
	lastNav := time.Now().UnixMilli()
	for {
		sessionID++
		steps, pauseMin, pauseMax, backChance := routeProfile(rng)
		events := make([]Event, 0, steps*2+1)
		for range steps {
			if !sleepBetween(ctx, rng, pauseMin, pauseMax) {
				return
			}
			state = s.snapshot()
			if !state.Ready || state.UserID == 0 {
				return
			}
			from, to := planner.next(rng, backChance)
			actionID = (actionID + 1) & 0xffffffff
			nav := NavigationEvent(state, sessionID, from, to, lastNav, actionID, rng)
			lastNav = nav.Time
			events = append(events, nav)
			if to == ScreenChat || (to == ScreenChats && rng.Float64() < s.timing.ChatsRenderChance) {
				if !sleepBetween(ctx, rng, s.timing.RenderMin, s.timing.RenderMax) {
					return
				}
				if to == ScreenChat {
					events = append(events, OpenChatEvent(state.UserID, sessionID, rng))
				} else {
					events = append(events, OpenChatsEvent(state.UserID, sessionID, rng))
				}
			}
		}
		if planner.current != ScreenBackground && rng.Float64() < s.timing.ReturnChance {
			if !sleepBetween(ctx, rng, s.timing.ReturnMin, s.timing.ReturnMax) {
				return
			}
			state = s.snapshot()
			if !state.Ready {
				return
			}
			actionID = (actionID + 1) & 0xffffffff
			nav := NavigationEvent(state, sessionID, planner.current, ScreenBackground, lastNav, actionID, rng)
			lastNav = nav.Time
			events = append(events, nav)
		}
		planner.reset()
		s.send(ctx, events)
		if !sleepBetween(ctx, rng, s.timing.SessionIdleMin, s.timing.SessionIdleMax) {
			return
		}
	}
}

func (s *Service) finish(done chan struct{}) {
	close(done)
	s.mu.Lock()
	if s.done == done {
		s.cancel, s.done = nil, nil
	}
	s.mu.Unlock()
}

func (s *Service) send(ctx context.Context, events []Event) {
	if len(events) == 0 || !s.snapshot().Ready {
		return
	}
	_, _ = s.invoker.Invoke(ctx, protocol.OpLog, map[string]any{"events": events})
}

// LoginEvent builds the initial performance event.
func LoginEvent(userID int64, sessionID int) Event {
	return Event{Time: time.Now().UnixMilli(), UserID: userID, Type: "PERF", Event: "login", SessionID: sessionID, Params: map[string]any{
		"properties": map[string]any{"connection_type": 2, "vpn": 0, "class": 2, "background": 1, "warm_start": 1},
		"errorType":  100,
	}}
}

func NavigationEvent(state Snapshot, sessionID int, from, to Screen, previous int64, actionID int, rng *rand.Rand) Event {
	params := map[string]any{"prev_time": previous, "screen_to": int(to), "action_id": actionID, "screen_from": int(from)}
	if to == ScreenChats {
		params["source_type"], params["source_id"], params["tab_config"] = 5, 1, 2
	} else if to == ScreenChat {
		params["source_type"], params["source_id"] = 1, state.UserID
		if len(state.Chats) > 0 {
			chat := state.Chats[rng.Intn(len(state.Chats))]
			params["source_id"] = chat.ID
			if chat.Type != types.ChatTypeDialog {
				params["source_type"] = 2
			}
		}
	}
	return Event{Time: time.Now().UnixMilli(), UserID: state.UserID, Type: "NAV", Event: "GO", Params: params, SessionID: sessionID}
}

func OpenChatEvent(userID int64, sessionID int, rng *rand.Rand) Event {
	messages, render := betweenInt(rng, 60, 240), betweenInt(rng, 50, 260)
	return Event{Time: time.Now().UnixMilli(), UserID: userID, Type: "PERF", Event: "open_chat_to_render", SessionID: sessionID, Params: map[string]any{
		"spans":      []map[string]any{{"duration": messages + render, "name": "open_chat_to_render"}, {"duration": messages, "name": "messages_list_created"}, {"duration": render, "name": "messages_render"}},
		"properties": map[string]any{"class": 2, "warm": 1, "flow": 1},
	}}
}

func OpenChatsEvent(userID int64, sessionID int, rng *rand.Rand) Event {
	created, rendered := betweenInt(rng, 50, 230), betweenInt(rng, 180, 650)
	return Event{Time: time.Now().UnixMilli(), UserID: userID, Type: "PERF", Event: "open_chats_to_render", SessionID: sessionID, Params: map[string]any{
		"spans":      []map[string]any{{"duration": created + rendered, "name": "open_chats_to_render"}, {"duration": created, "name": "chats_tab_created"}, {"duration": rendered, "name": "chat_list_render"}},
		"properties": map[string]any{"class": 2},
	}}
}

type planner struct {
	current Screen
	history []Screen
}

func newPlanner() *planner { return &planner{current: ScreenBackground} }

func (p *planner) reset() { p.current, p.history = ScreenBackground, nil }

func (p *planner) next(rng *rand.Rand, backChance float64) (Screen, Screen) {
	from := p.current
	if len(p.history) > 0 && rng.Float64() < backChance {
		p.current = p.history[len(p.history)-1]
		p.history = p.history[:len(p.history)-1]
		return from, p.current
	}
	choices := transitions[p.current]
	total := 0
	for _, choice := range choices {
		total += choice.weight
	}
	point, next := rng.Intn(total)+1, choices[len(choices)-1].screen
	for _, choice := range choices {
		point -= choice.weight
		if point <= 0 {
			next = choice.screen
			break
		}
	}
	if next != p.current {
		p.history = append(p.history, p.current)
		if len(p.history) > 4 {
			p.history = p.history[1:]
		}
	}
	p.current = next
	return from, next
}

type transition struct {
	screen Screen
	weight int
}

var transitions = map[Screen][]transition{
	ScreenBackground: {{ScreenChats, 10}, {ScreenSettings, 1}},
	ScreenChats:      {{ScreenChat, 7}, {ScreenContacts, 2}, {ScreenSearch, 2}, {ScreenCalls, 1}, {ScreenSettings, 1}, {ScreenChats, 2}},
	ScreenChat:       {{ScreenChats, 8}, {ScreenChat, 2}, {ScreenSettings, 1}},
	ScreenContacts:   {{ScreenChats, 6}, {ScreenChat, 2}, {ScreenSearch, 1}},
	ScreenSearch:     {{ScreenChats, 5}, {ScreenChat, 3}, {ScreenContacts, 1}},
	ScreenCalls:      {{ScreenChats, 5}, {ScreenContacts, 2}, {ScreenSettings, 2}},
	ScreenSettings:   {{ScreenChats, 7}, {ScreenContacts, 2}, {ScreenCalls, 2}, {ScreenMiniApp, 1}},
	ScreenMiniApp:    {{ScreenSettings, 3}, {ScreenChats, 6}},
}

func routeProfile(rng *rand.Rand) (int, time.Duration, time.Duration, float64) {
	switch rng.Intn(3) {
	case 0:
		return 2, 35 * time.Second, 95 * time.Second, .30
	case 1:
		return 4, 70 * time.Second, 210 * time.Second, .22
	default:
		return 3, 140 * time.Second, 360 * time.Second, .18
	}
}

func normalizeTiming(value Timing) Timing {
	defaults := DefaultTiming
	if value.StartupMin == 0 {
		value.StartupMin = defaults.StartupMin
	}
	if value.StartupMax == 0 {
		value.StartupMax = defaults.StartupMax
	}
	if value.SessionIdleMin == 0 {
		value.SessionIdleMin = defaults.SessionIdleMin
	}
	if value.SessionIdleMax == 0 {
		value.SessionIdleMax = defaults.SessionIdleMax
	}
	if value.RenderMin == 0 {
		value.RenderMin = defaults.RenderMin
	}
	if value.RenderMax == 0 {
		value.RenderMax = defaults.RenderMax
	}
	if value.ReturnMin == 0 {
		value.ReturnMin = defaults.ReturnMin
	}
	if value.ReturnMax == 0 {
		value.ReturnMax = defaults.ReturnMax
	}
	if value.ReturnChance == 0 {
		value.ReturnChance = defaults.ReturnChance
	}
	if value.ChatsRenderChance == 0 {
		value.ChatsRenderChance = defaults.ChatsRenderChance
	}
	return value
}

func sleepBetween(ctx context.Context, rng *rand.Rand, min, max time.Duration) bool {
	if max < min {
		min, max = max, min
	}
	delay := min
	if max > min {
		delay += time.Duration(rng.Int63n(int64(max - min)))
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func betweenInt(rng *rand.Rand, min, max int) int { return min + rng.Intn(max-min+1) }

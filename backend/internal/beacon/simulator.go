package beacon

import "sync"

type Simulator struct {
	mu          sync.RWMutex
	lastPayload *Payload
}

func NewSimulator() *Simulator {
	return &Simulator{}
}

func (s *Simulator) Update(payload Payload) (Status, error) {
	if err := payload.Validate(); err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := payload
	s.lastPayload = &copy
	return s.statusLocked(), nil
}

func (s *Simulator) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *Simulator) statusLocked() Status {
	var lastPayload *Payload
	if s.lastPayload != nil {
		copy := *s.lastPayload
		lastPayload = &copy
	}
	return Status{
		Status: "simulated", Mode: "simulation",
		Reason:      "Virtual output only; no physical serial device is opened.",
		LastPayload: lastPayload,
	}
}

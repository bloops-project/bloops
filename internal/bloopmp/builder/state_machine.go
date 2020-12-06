package builder

import "container/list"

func newStateMachine(kind ...stateKind) *stateMachine {
	machine := &stateMachine{
		transitions: list.New(),
	}

	if len(kind) > 0 {
		machine.max = kind[len(kind)-1]
		machine.min = kind[0]

		for i := range kind {
			machine.transitions.PushBack(kind[i])
		}

		machine.transitions.PushFront(stateKindLetters)
		machine.transitions.PushFront(stateKindRounds)
		machine.transitions.PushFront(stateKindCategories)

		machine.front()
	}

	return machine
}

type stateMachine struct {
	min, max    stateKind
	state       stateKind
	transitions *list.List
}

func (s *stateMachine) curr() stateKind {
	return s.state
}

func (s *stateMachine) back() {
	s.state = s.transitions.Back().Value.(stateKind)
}

func (s *stateMachine) front() {
	s.state = s.transitions.Front().Value.(stateKind)
}

func (s *stateMachine) isMax() bool {
	return s.state == s.max
}

func (s *stateMachine) isMin() bool {
	return s.state == s.min
}

func (s *stateMachine) seek(kind stateKind) {
	for e := s.transitions.Front(); e != nil; e = e.Next() {
		if e.Value == kind {
			s.state = kind
			break
		}
	}
}

func (s *stateMachine) next() bool {
	if s.isMax() {
		return false
	}

	for e := s.transitions.Front(); e != nil; e = e.Next() {
		if e.Value == s.state {
			s.state = e.Next().Value.(stateKind)
			break
		}
	}

	return true
}

func (s *stateMachine) prev() bool {
	if s.isMin() {
		return false
	}
	for e := s.transitions.Front(); e != nil; e = e.Next() {
		if e.Value == s.state {
			s.state = e.Prev().Value.(stateKind)
			break
		}
	}

	return true
}

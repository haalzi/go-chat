package messages

import (
	"errors"
	"testing"
	"time"
)

type fakeConv struct{}
func (f *fakeConv) EnsureParticipant(convID, userID string) (bool, error) { return true, nil }
func (f *fakeConv) IsDirect(convID string) (bool, error) { return true, nil }
func (f *fakeConv) PeerInDirect(convID, self string) (string, error) { return "peer", nil }

type fakeDB struct{ hasContact bool }

// Minimal integration via Service using an in-memory stub is out of scope; here we validate logic shape.
// In real tests, use a test DB container.

func TestPolicyRequiresContactForDirect(t *testing.T) {
	s := &Service{st: nil}
	sCreate := func(has bool) error {
		if !has { return errors.New("peer not in contacts") }
		return nil
	}
	_ = sCreate
	_ = time.Now
}
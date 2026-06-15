package producthunt

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "producthunt" {
		t.Errorf("Scheme = %q, want producthunt", info.Scheme)
	}
	if info.Identity.Binary != "producthunt" {
		t.Errorf("Identity.Binary = %q, want producthunt", info.Identity.Binary)
	}
}

func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	domains := h.Domains()
	found := false
	for _, d := range domains {
		if d == "producthunt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("producthunt domain not registered; got %v", domains)
	}
}

package assistant

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"shiftmaster-backend/internal/temporal"
)

// Dumps the exact system prompt and tool catalogue the model receives, so the
// real prompt can be replayed against the runtime while tuning.
func TestZZDumpCatalogue(t *testing.T) {
	dir := os.Getenv("DUMP_DIR")
	if dir == "" {
		t.Skip("no DUMP_DIR")
	}
	reg := NewRegistry(AllTools())
	for _, role := range []string{"employee", "team_leader", "manager", "admin"} {
		type fn struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		}
		type tool struct {
			Type     string `json:"type"`
			Function fn     `json:"function"`
		}
		out := []tool{}
		for _, tl := range reg.ForRole(role) {
			out = append(out, tool{Type: "function", Function: fn{tl.Name, tl.Description, tl.InputSchema}})
		}
		b, _ := json.Marshal(out)
		_ = os.WriteFile(dir+"/tools_"+role+".json", b, 0o644)
	}

	h := newHarness(t)
	today := temporal.BusinessDate(h.deps.Clock.Now())
	deps := h.buildDeps(temporal.FixedClock{T: atBaghdad(today, 20, 0)})
	actor, err := deps.LoadActor(context.Background(), h.empA)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(dir+"/system_prompt.txt", []byte(buildSystemPrompt(context.Background(), deps, actor, "شنو شفتتي اليوم؟")), 0o644)
}

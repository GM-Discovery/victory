package commands

import (
	"context"
	"testing"
)

func TestExecuteIdempotentRequiresActor(t *testing.T) {
	_, _, err := ExecuteIdempotent(context.Background(), nil, "", "char.set.name", "key-1", func(ctx context.Context) (map[string]any, error) {
		t.Fatalf("fn should not be called when actor is missing")
		return nil, nil
	})
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestExecuteIdempotentWithoutKeyExecutesDirectly(t *testing.T) {
	calls := 0
	result, replayed, err := ExecuteIdempotent(context.Background(), nil, "user-1", "char.set.name", "", func(ctx context.Context) (map[string]any, error) {
		calls++
		return map[string]any{"ok": true}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if replayed {
		t.Fatalf("expected replayed=false when no idempotency key supplied")
	}
	if calls != 1 {
		t.Fatalf("expected fn to be called exactly once, got %d", calls)
	}
	if result["ok"] != true {
		t.Fatalf("unexpected result: %+v", result)
	}
}

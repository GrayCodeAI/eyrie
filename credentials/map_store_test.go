package credentials

import (
	"context"
	"testing"
)

func TestMapStore_SetGetDelete(t *testing.T) {
	tests := []struct {
		name      string
		setKey    string
		setVal    string
		getKey    string
		wantVal   string
		wantErr   bool
		deleteKey string
	}{
		{
			name:    "basic_set_get",
			setKey:  "anthropic_api_key",
			setVal:  "sk-ant-123",
			getKey:  "anthropic_api_key",
			wantVal: "sk-ant-123",
		},
		{
			name:    "case_insensitive_get",
			setKey:  "openai_api_key",
			setVal:  "sk-test",
			getKey:  "OPENAI_API_KEY",
			wantVal: "sk-test",
		},
		{
			name:    "trimmed_key_lookup",
			setKey:  "  gemini_api_key  ",
			setVal:  "key-val",
			getKey:  "gemini_api_key",
			wantVal: "key-val",
		},
		{
			name:    "get_missing_key_returns_not_found",
			setKey:  "some_key",
			setVal:  "val",
			getKey:  "nonexistent",
			wantErr: true,
		},
		{
			name:    "whitespace_only_value_treated_as_missing",
			setKey:  "ws_key",
			setVal:  "   ",
			getKey:  "ws_key",
			wantErr: true,
		},
		{
			name:      "delete_removes_key",
			setKey:    "del_key",
			setVal:    "del_val",
			getKey:    "del_key",
			deleteKey: "del_key",
			wantErr:   true,
		},
		{
			name:      "delete_nonexistent_is_noop",
			deleteKey: "never_existed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MapStore{}
			ctx := context.Background()

			if tt.setKey != "" {
				if err := ms.Set(ctx, tt.setKey, tt.setVal); err != nil {
					t.Fatalf("Set(%q, %q) error: %v", tt.setKey, tt.setVal, err)
				}
			}

			if tt.deleteKey != "" {
				if err := ms.Delete(ctx, tt.deleteKey); err != nil {
					t.Fatalf("Delete(%q) error: %v", tt.deleteKey, err)
				}
			}

			if tt.getKey != "" {
				got, err := ms.Get(ctx, tt.getKey)
				if tt.wantErr {
					if err == nil {
						t.Fatalf("Get(%q) = %q, want error", tt.getKey, got)
					}
					if err != ErrNotFound {
						t.Fatalf("Get(%q) error = %v, want ErrNotFound", tt.getKey, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("Get(%q) error: %v", tt.getKey, err)
				}
				if got != tt.wantVal {
					t.Errorf("Get(%q) = %q, want %q", tt.getKey, got, tt.wantVal)
				}
			}
		})
	}
}

func TestMapStore_GetNilData(t *testing.T) {
	ms := &MapStore{}
	_, err := ms.Get(context.Background(), "any_key")
	if err != ErrNotFound {
		t.Fatalf("Get on empty MapStore: err = %v, want ErrNotFound", err)
	}
}

func TestMapStore_DeleteNilData(t *testing.T) {
	ms := &MapStore{}
	// Should not panic.
	if err := ms.Delete(context.Background(), "any_key"); err != nil {
		t.Fatalf("Delete on empty MapStore: err = %v", err)
	}
}

func TestMapStore_Overwrite(t *testing.T) {
	ms := &MapStore{}
	ctx := context.Background()

	if err := ms.Set(ctx, "key", "first"); err != nil {
		t.Fatal(err)
	}
	if err := ms.Set(ctx, "key", "second"); err != nil {
		t.Fatal(err)
	}
	got, err := ms.Get(ctx, "key")
	if err != nil || got != "second" {
		t.Fatalf("after overwrite: Get = %q, err = %v, want %q", got, err, "second")
	}
}

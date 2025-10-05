package main

import (
	"testing"

	c "github.com/imjamesonzeller/tasklight-v3/config"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
)

func TestParseTaskFromContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    TaskInformation
		wantErr bool
	}{
		{
			name:  "valid with date",
			input: `{"title":"Write report","date":"2024-10-01"}`,
			want: TaskInformation{
				Title: "Write report",
				Date:  ptr("2024-10-01"),
			},
		},
		{
			name:  "valid without date",
			input: `{"title":"Buy milk","date":null}`,
			want:  TaskInformation{Title: "Buy milk", Date: nil},
		},
		{
			name:    "invalid json",
			input:   `{"title":"oops"`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseTaskFromContent(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Title != tc.want.Title {
				t.Errorf("title mismatch: got %q want %q", got.Title, tc.want.Title)
			}

			switch {
			case tc.want.Date == nil && got.Date != nil:
				t.Fatalf("expected nil date, got %v", *got.Date)
			case tc.want.Date != nil && got.Date == nil:
				t.Fatalf("expected date %v, got nil", *tc.want.Date)
			case tc.want.Date != nil && got.Date != nil && *tc.want.Date != *got.Date:
				t.Fatalf("date mismatch: got %v want %v", *got.Date, *tc.want.Date)
			}
		})
	}
}

func TestBuildNotionPagePayload(t *testing.T) {
	t.Parallel()

	originalConfig := c.AppConfig
	t.Cleanup(func() { c.AppConfig = originalConfig })

	date := "2024-10-02"
	c.AppConfig = &settingsservice.ApplicationSettings{
		NotionDBID:       "db-123",
		DatePropertyName: "Due",
	}

	payload := buildNotionPagePayload(TaskInformation{Title: "Plan", Date: &date})

	parent, ok := payload["parent"].(map[string]any)
	if !ok {
		t.Fatalf("parent field missing or wrong type")
	}
	if parent["database_id"] != "db-123" {
		t.Fatalf("unexpected database id: %v", parent["database_id"])
	}

	props, ok := payload["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing")
	}

	nameProp, ok := props["Name"].(map[string]any)
	if !ok {
		t.Fatalf("name property missing")
	}

	titleSlice, ok := nameProp["title"].([]map[string]any)
	if !ok {
		t.Fatalf("title slice missing")
	}
	text, ok := titleSlice[0]["text"].(map[string]any)
	if !ok || text["content"] != "Plan" {
		t.Fatalf("unexpected title content: %v", titleSlice)
	}

	dueProp, ok := props["Due"].(map[string]any)
	if !ok {
		t.Fatalf("date property missing")
	}
	dateMap, ok := dueProp["date"].(map[string]any)
	if !ok || dateMap["start"] != date {
		t.Fatalf("unexpected date payload: %v", dueProp)
	}
}

func ptr[T any](v T) *T {
	return &v
}

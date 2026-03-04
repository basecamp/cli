package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Exit Codes Tests
// =============================================================================

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{CodeUsage, ExitUsage},
		{CodeNotFound, ExitNotFound},
		{CodeAuth, ExitAuth},
		{CodeForbidden, ExitForbidden},
		{CodeRateLimit, ExitRateLimit},
		{CodeNetwork, ExitNetwork},
		{CodeAPI, ExitAPI},
		{CodeAmbiguous, ExitAmbiguous},
		{"unknown_code", ExitAPI}, // Unknown codes default to ExitAPI
		{"", ExitAPI},             // Empty code defaults to ExitAPI
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := ExitCodeFor(tt.code)
			assert.Equal(t, tt.expected, result, "ExitCodeFor(%q)", tt.code)
		})
	}
}

func TestExitCodeConstants(t *testing.T) {
	expected := map[int]int{
		ExitOK:        0,
		ExitUsage:     1,
		ExitNotFound:  2,
		ExitAuth:      3,
		ExitForbidden: 4,
		ExitRateLimit: 5,
		ExitNetwork:   6,
		ExitAPI:       7,
		ExitAmbiguous: 8,
	}

	for code, value := range expected {
		assert.Equal(t, value, code, "Exit code constant mismatch")
	}
}

func TestErrorCodeConstants(t *testing.T) {
	codes := []string{
		CodeUsage,
		CodeNotFound,
		CodeAuth,
		CodeForbidden,
		CodeRateLimit,
		CodeNetwork,
		CodeAPI,
		CodeAmbiguous,
	}

	for _, code := range codes {
		assert.NotEmpty(t, code, "Error code should not be empty")
	}
}

// =============================================================================
// Error Struct Tests
// =============================================================================

func TestErrorInterface(t *testing.T) {
	errWithHint := &Error{
		Code:    CodeNotFound,
		Message: "resource not found",
		Hint:    "check the ID",
	}
	assert.Equal(t, "resource not found: check the ID", errWithHint.Error())

	errNoHint := &Error{
		Code:    CodeNotFound,
		Message: "resource not found",
	}
	assert.Equal(t, "resource not found", errNoHint.Error())
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &Error{
		Code:    CodeAPI,
		Message: "api error",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped) //nolint:errorlint // testing Unwrap returns exact wrapped error
}

func TestErrorUnwrapNil(t *testing.T) {
	err := &Error{
		Code:    CodeAPI,
		Message: "api error",
	}

	assert.Nil(t, err.Unwrap(), "Unwrap() should return nil when Cause is nil")
}

func TestErrorExitCode(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{CodeUsage, ExitUsage},
		{CodeNotFound, ExitNotFound},
		{CodeAuth, ExitAuth},
		{CodeForbidden, ExitForbidden},
		{CodeRateLimit, ExitRateLimit},
		{CodeNetwork, ExitNetwork},
		{CodeAPI, ExitAPI},
		{CodeAmbiguous, ExitAmbiguous},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := &Error{Code: tt.code, Message: "test"}
			assert.Equal(t, tt.expected, err.ExitCode())
		})
	}
}

// =============================================================================
// Error Constructors Tests
// =============================================================================

func TestErrUsage(t *testing.T) {
	err := ErrUsage("invalid argument")

	assert.Equal(t, CodeUsage, err.Code)
	assert.Equal(t, "invalid argument", err.Message)
	assert.Equal(t, ExitUsage, err.ExitCode())
}

func TestErrUsageHint(t *testing.T) {
	err := ErrUsageHint("invalid argument", "try --help")

	assert.Equal(t, CodeUsage, err.Code)
	assert.Equal(t, "invalid argument", err.Message)
	assert.Equal(t, "try --help", err.Hint)
}

func TestErrNotFound(t *testing.T) {
	err := ErrNotFound("project", "123")

	assert.Equal(t, CodeNotFound, err.Code)
	assert.Equal(t, "project not found: 123", err.Message)
	assert.Equal(t, ExitNotFound, err.ExitCode())
}

func TestErrNotFoundHint(t *testing.T) {
	err := ErrNotFoundHint("project", "123", "check project ID")

	assert.Equal(t, CodeNotFound, err.Code)
	assert.Equal(t, "check project ID", err.Hint)
}

func TestErrAuth(t *testing.T) {
	err := ErrAuth("not authenticated")

	assert.Equal(t, CodeAuth, err.Code)
	assert.NotEmpty(t, err.Hint, "Hint should contain login instruction")
	assert.Equal(t, ExitAuth, err.ExitCode())
}

func TestErrForbidden(t *testing.T) {
	err := ErrForbidden("access denied")

	assert.Equal(t, CodeForbidden, err.Code)
	assert.Equal(t, 403, err.HTTPStatus)
	assert.Equal(t, ExitForbidden, err.ExitCode())
}

func TestErrForbiddenScope(t *testing.T) {
	err := ErrForbiddenScope()

	assert.Equal(t, CodeForbidden, err.Code)
	assert.Equal(t, 403, err.HTTPStatus)
	assert.NotEmpty(t, err.Hint, "Hint should not be empty for scope error")
}

func TestErrRateLimit(t *testing.T) {
	err := ErrRateLimit(60)

	assert.Equal(t, CodeRateLimit, err.Code)
	assert.Equal(t, 429, err.HTTPStatus)
	assert.True(t, err.Retryable, "RateLimit error should be retryable")
	assert.NotEmpty(t, err.Hint, "Hint should contain retry time")
	assert.Equal(t, ExitRateLimit, err.ExitCode())
}

func TestErrRateLimitZero(t *testing.T) {
	err := ErrRateLimit(0)

	assert.Equal(t, "Try again later", err.Hint)
}

func TestErrNetwork(t *testing.T) {
	cause := errors.New("connection refused")
	err := ErrNetwork(cause)

	assert.Equal(t, CodeNetwork, err.Code)
	assert.True(t, err.Retryable, "Network error should be retryable")
	assert.Equal(t, cause, err.Cause) //nolint:errorlint // testing Cause field is exact wrapped error
	assert.Equal(t, "connection refused", err.Hint)
	assert.Equal(t, ExitNetwork, err.ExitCode())
}

func TestErrAPI(t *testing.T) {
	err := ErrAPI(500, "server error")

	assert.Equal(t, CodeAPI, err.Code)
	assert.Equal(t, 500, err.HTTPStatus)
	assert.Equal(t, "server error", err.Message)
	assert.Equal(t, ExitAPI, err.ExitCode())
}

func TestErrAmbiguous(t *testing.T) {
	matches := []string{"Project A", "Project B", "Project Alpha"}
	err := ErrAmbiguous("multiple matches", matches)

	assert.Equal(t, CodeAmbiguous, err.Code)
	assert.NotEmpty(t, err.Hint, "Hint should contain matches")
	assert.Equal(t, ExitAmbiguous, err.ExitCode())
}

// =============================================================================
// AsError Tests
// =============================================================================

func TestAsErrorWithOutputError(t *testing.T) {
	original := &Error{
		Code:    CodeNotFound,
		Message: "not found",
		Hint:    "try again",
	}

	result := AsError(original)
	assert.Equal(t, original, result, "AsError should return same *Error unchanged")
}

func TestAsErrorWithStandardError(t *testing.T) {
	original := errors.New("something went wrong")

	result := AsError(original)
	assert.Equal(t, CodeAPI, result.Code)
	assert.Equal(t, "something went wrong", result.Message)
	assert.Equal(t, original, result.Cause) //nolint:errorlint // testing Cause field is exact original error
}

func TestAsErrorWithWrappedOutputError(t *testing.T) {
	original := &Error{
		Code:    CodeAuth,
		Message: "auth required",
	}
	wrapped := errors.Join(errors.New("wrapper"), original)

	result := AsError(wrapped)
	assert.Equal(t, CodeAuth, result.Code)
}

// =============================================================================
// Envelope/Response Tests
// =============================================================================

func TestResponseJSON(t *testing.T) {
	resp := &Response{
		OK:      true,
		Data:    map[string]string{"name": "Test Project"},
		Summary: "Found 1 project",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err, "Failed to marshal")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded), "Failed to unmarshal")

	assert.Equal(t, true, decoded["ok"])
	assert.Equal(t, "Found 1 project", decoded["summary"])
}

func TestErrorResponseJSON(t *testing.T) {
	resp := &ErrorResponse{
		OK:    false,
		Error: "not found",
		Code:  CodeNotFound,
		Hint:  "check the ID",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err, "Failed to marshal")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded), "Failed to unmarshal")

	assert.Equal(t, false, decoded["ok"])
	assert.Equal(t, "not found", decoded["error"])
	assert.Equal(t, CodeNotFound, decoded["code"])
}

func TestBreadcrumb(t *testing.T) {
	bc := Breadcrumb{
		Action:      "show",
		Cmd:         "mycli projects show 123",
		Description: "View project details",
	}

	data, err := json.Marshal(bc)
	require.NoError(t, err, "Failed to marshal")

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(data, &decoded), "Failed to unmarshal")

	assert.Equal(t, "show", decoded["action"])
	assert.Equal(t, "mycli projects show 123", decoded["cmd"])
}

// =============================================================================
// Writer Tests
// =============================================================================

func TestWriterOK(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatJSON,
		Writer: &buf,
	})

	data := map[string]string{"id": "123", "name": "Test"}
	err := w.OK(data, WithSummary("test summary"))
	require.NoError(t, err, "OK() failed")

	var resp Response
	require.NoError(t, json.Unmarshal(buf.Bytes(), &resp), "Failed to unmarshal output")

	assert.True(t, resp.OK)
	assert.Equal(t, "test summary", resp.Summary)
}

func TestWriterErr(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatJSON,
		Writer: &buf,
	})

	err := w.Err(ErrNotFound("project", "123"))
	require.NoError(t, err, "Err() failed")

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &resp), "Failed to unmarshal output")

	assert.False(t, resp.OK)
	assert.Equal(t, CodeNotFound, resp.Code)
}

func TestWriterQuietFormat(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatQuiet,
		Writer: &buf,
	})

	data := map[string]string{"id": "123", "name": "Test"}
	err := w.OK(data, WithSummary("ignored"))
	require.NoError(t, err, "OK() failed")

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded), "Failed to unmarshal output")

	assert.Equal(t, "123", decoded["id"])
	_, exists := decoded["ok"]
	assert.False(t, exists, "Quiet format should not include envelope ok field")
}

func TestWriterQuietFormatString(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatQuiet,
		Writer: &buf,
	})

	err := w.OK("my-auth-token-value")
	require.NoError(t, err, "OK() failed")

	output := buf.String()
	assert.Equal(t, "\"my-auth-token-value\"\n", output)
}

func TestWriterIDsFormat(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatIDs,
		Writer: &buf,
	})

	data := []map[string]any{
		{"id": 123, "name": "Project A"},
		{"id": 456, "name": "Project B"},
	}
	err := w.OK(data)
	require.NoError(t, err, "OK() failed")

	output := buf.String()
	assert.Equal(t, "123\n456\n", output)
}

func TestWriterIDsFormatWithSingleItem(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatIDs,
		Writer: &buf,
	})

	data := map[string]any{"id": 999, "name": "Single"}
	err := w.OK(data)
	require.NoError(t, err, "OK() failed")

	output := buf.String()
	assert.Equal(t, "999\n", output)
}

func TestWriterIDsFormatWithNoID(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatIDs,
		Writer: &buf,
	})

	data := []map[string]any{
		{"name": "No ID"},
	}
	err := w.OK(data)
	require.NoError(t, err, "OK() failed")

	output := buf.String()
	assert.Equal(t, "", output)
}

func TestWriterCountFormat(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatCount,
		Writer: &buf,
	})

	data := []map[string]any{
		{"id": 1},
		{"id": 2},
		{"id": 3},
	}
	err := w.OK(data)
	require.NoError(t, err, "OK() failed")

	output := buf.String()
	assert.Equal(t, "3\n", output)
}

func TestWriterCountFormatSingleItem(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatCount,
		Writer: &buf,
	})

	data := map[string]any{"id": 1, "name": "Single"}
	err := w.OK(data)
	require.NoError(t, err, "OK() failed")

	output := buf.String()
	assert.Equal(t, "1\n", output)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, FormatAuto, opts.Format)
	assert.NotNil(t, opts.Writer, "Default Writer should not be nil")
}

func TestNewWithNilWriter(t *testing.T) {
	w := New(Options{
		Format: FormatJSON,
		Writer: nil,
	})

	assert.NotNil(t, w.opts.Writer, "Writer should default to os.Stdout, not nil")
}

// =============================================================================
// Response Options Tests
// =============================================================================

func TestWithSummary(t *testing.T) {
	resp := &Response{}
	WithSummary("test summary")(resp)

	assert.Equal(t, "test summary", resp.Summary)
}

func TestWithNotice(t *testing.T) {
	resp := &Response{}
	WithNotice("truncated")(resp)

	assert.Equal(t, "truncated", resp.Notice)
}

func TestWithBreadcrumbs(t *testing.T) {
	resp := &Response{}
	bc1 := Breadcrumb{Action: "list", Cmd: "mycli list", Description: "List items"}
	bc2 := Breadcrumb{Action: "show", Cmd: "mycli show 1", Description: "Show item"}

	WithBreadcrumbs(bc1, bc2)(resp)

	require.Len(t, resp.Breadcrumbs, 2)
	assert.Equal(t, "list", resp.Breadcrumbs[0].Action)
}

func TestWithBreadcrumbsAppend(t *testing.T) {
	resp := &Response{
		Breadcrumbs: []Breadcrumb{{Action: "initial"}},
	}
	bc := Breadcrumb{Action: "added"}

	WithBreadcrumbs(bc)(resp)

	assert.Len(t, resp.Breadcrumbs, 2)
}

func TestWithoutBreadcrumbs(t *testing.T) {
	resp := &Response{
		Breadcrumbs: []Breadcrumb{{Action: "existing"}},
	}

	WithoutBreadcrumbs()(resp)

	assert.Nil(t, resp.Breadcrumbs)
}

func TestWithContext(t *testing.T) {
	resp := &Response{}

	WithContext("project_id", 123)(resp)
	WithContext("user", "alice")(resp)

	assert.Equal(t, 123, resp.Context["project_id"])
	assert.Equal(t, "alice", resp.Context["user"])
}

func TestWithMeta(t *testing.T) {
	resp := &Response{}

	WithMeta("page", 1)(resp)
	WithMeta("total", 100)(resp)

	assert.Equal(t, 1, resp.Meta["page"])
	assert.Equal(t, 100, resp.Meta["total"])
}

// =============================================================================
// NormalizeData Tests
// =============================================================================

func TestNormalizeDataWithSlice(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "name": "A"},
		{"id": 2, "name": "B"},
	}

	result := NormalizeData(data)
	slice, ok := result.([]map[string]any)
	require.True(t, ok, "Expected []map[string]any, got %T", result)
	assert.Len(t, slice, 2)
}

func TestNormalizeDataWithMap(t *testing.T) {
	data := map[string]any{"id": 1, "name": "A"}

	result := NormalizeData(data)
	m, ok := result.(map[string]any)
	require.True(t, ok, "Expected map[string]any, got %T", result)
	assert.Equal(t, 1, m["id"])
}

func TestNormalizeDataWithJSONRawMessage(t *testing.T) {
	raw := json.RawMessage(`[{"id": 1}, {"id": 2}]`)

	result := NormalizeData(raw)
	slice, ok := result.([]map[string]any)
	require.True(t, ok, "Expected []map[string]any, got %T", result)
	assert.Len(t, slice, 2)
}

func TestNormalizeDataWithStruct(t *testing.T) {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := Item{ID: 1, Name: "Test"}

	result := NormalizeData(data)
	m, ok := result.(map[string]any)
	require.True(t, ok, "Expected map[string]any, got %T", result)
	assert.Equal(t, json.Number("1"), m["id"]) // UseNumber preserves numeric precision
}

func TestNormalizeDataWithNil(t *testing.T) {
	result := NormalizeData(nil)
	assert.Nil(t, result)
}

func TestNormalizeDataPreservesLargeIDs(t *testing.T) {
	// Verify json.Number preservation for large IDs that exceed float64 precision
	raw := json.RawMessage(`{"id": 9007199254740993}`)

	result := NormalizeData(raw)
	m, ok := result.(map[string]any)
	require.True(t, ok, "Expected map[string]any, got %T", result)

	id, ok := m["id"].(json.Number)
	require.True(t, ok, "Expected json.Number, got %T", m["id"])
	assert.Equal(t, "9007199254740993", id.String())
}

func TestNormalizeDataEmptyArray(t *testing.T) {
	raw := json.RawMessage(`[]`)

	result := NormalizeData(raw)
	slice, ok := result.([]map[string]any)
	require.True(t, ok, "Expected []map[string]any, got %T", result)
	assert.Len(t, slice, 0)
}

// =============================================================================
// TruncationNotice Tests
// =============================================================================

func TestTruncationNotice(t *testing.T) {
	tests := []struct {
		name          string
		count         int
		defaultLimit  int
		all           bool
		explicitLimit int
		expected      string
	}{
		{"at limit", 100, 100, false, 0, "Showing 100 results (use --all for complete list)"},
		{"below limit", 50, 100, false, 0, ""},
		{"with --all", 100, 100, true, 0, ""},
		{"zero limit", 50, 0, false, 0, ""},
		{"explicit limit at boundary", 25, 100, false, 25, "Showing 25 results (use --all for complete list)"},
		{"explicit limit not reached", 10, 100, false, 25, ""},
		{"above limit", 150, 100, false, 0, "Showing 150 results (use --all for complete list)"},
		{"zero count", 0, 100, false, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncationNotice(tt.count, tt.defaultLimit, tt.all, tt.explicitLimit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncationNoticeWithTotal(t *testing.T) {
	tests := []struct {
		name       string
		count      int
		totalCount int
		expected   string
	}{
		{"truncated", 25, 100, "Showing 25 of 100 results (use --all for complete list)"},
		{"not truncated", 100, 100, ""},
		{"zero total", 25, 0, ""},
		{"count exceeds total", 100, 50, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncationNoticeWithTotal(tt.count, tt.totalCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// EffectiveFormat Tests
// =============================================================================

func TestEffectiveFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   Format
		expected Format
	}{
		{"JSON stays JSON", FormatJSON, FormatJSON},
		{"Markdown stays Markdown", FormatMarkdown, FormatMarkdown},
		{"Styled stays Styled", FormatStyled, FormatStyled},
		{"Quiet stays Quiet", FormatQuiet, FormatQuiet},
		{"IDs stays IDs", FormatIDs, FormatIDs},
		{"Count stays Count", FormatCount, FormatCount},
		// FormatAuto resolves to FormatJSON when writer is not a TTY (bytes.Buffer)
		{"Auto resolves to JSON for non-TTY", FormatAuto, FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := New(Options{
				Format: tt.format,
				Writer: &buf,
			})

			got := w.EffectiveFormat()
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// Format Constants Tests
// =============================================================================

func TestFormatConstants(t *testing.T) {
	formats := map[Format]string{
		FormatAuto:     "auto",
		FormatJSON:     "json",
		FormatMarkdown: "markdown",
		FormatStyled:   "styled",
		FormatQuiet:    "quiet",
		FormatIDs:      "ids",
		FormatCount:    "count",
	}

	seen := make(map[Format]bool)
	for format := range formats {
		assert.False(t, seen[format], "Duplicate format value: %d", format)
		seen[format] = true
	}
}

// =============================================================================
// Error Edge Case Tests
// =============================================================================

func TestErrorWithHTTPStatus(t *testing.T) {
	testCases := []struct {
		name           string
		err            *Error
		expectedStatus int
	}{
		{"forbidden", ErrForbidden("x"), 403},
		{"forbidden scope", ErrForbiddenScope(), 403},
		{"rate limit", ErrRateLimit(60), 429},
		{"api error", ErrAPI(500, "x"), 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStatus, tc.err.HTTPStatus)
		})
	}
}

func TestErrorRetryable(t *testing.T) {
	retryable := []struct {
		name string
		err  *Error
	}{
		{"rate limit", ErrRateLimit(60)},
		{"network", ErrNetwork(errors.New("connection failed"))},
	}

	for _, tc := range retryable {
		t.Run(tc.name+" is retryable", func(t *testing.T) {
			assert.True(t, tc.err.Retryable, "Expected error to be retryable")
		})
	}

	nonRetryable := []struct {
		name string
		err  *Error
	}{
		{"not found", ErrNotFound("x", "y")},
		{"auth", ErrAuth("x")},
		{"forbidden", ErrForbidden("x")},
		{"usage", ErrUsage("x")},
		{"ambiguous", ErrAmbiguous("x", nil)},
	}

	for _, tc := range nonRetryable {
		t.Run(tc.name+" is not retryable", func(t *testing.T) {
			assert.False(t, tc.err.Retryable, "Expected error not to be retryable")
		})
	}
}

// =============================================================================
// Writer with Styled/Markdown falls back to JSON
// =============================================================================

func TestWriterStyledFallsBackToJSON(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatStyled,
		Writer: &buf,
	})

	data := map[string]string{"id": "1", "name": "Test"}
	err := w.OK(data)
	require.NoError(t, err)

	// Styled/Markdown fall back to JSON in the shared package
	var resp Response
	require.NoError(t, json.Unmarshal(buf.Bytes(), &resp))
	assert.True(t, resp.OK)
}

func TestWriterMarkdownFallsBackToJSON(t *testing.T) {
	var buf bytes.Buffer
	w := New(Options{
		Format: FormatMarkdown,
		Writer: &buf,
	})

	data := map[string]string{"id": "1", "name": "Test"}
	err := w.OK(data)
	require.NoError(t, err)

	var resp Response
	require.NoError(t, json.Unmarshal(buf.Bytes(), &resp))
	assert.True(t, resp.OK)
}

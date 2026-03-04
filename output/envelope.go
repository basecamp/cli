package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Response is the success envelope for JSON output.
type Response struct {
	OK          bool           `json:"ok"`
	Data        any            `json:"data,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Notice      string         `json:"notice,omitempty"`
	Breadcrumbs []Breadcrumb   `json:"breadcrumbs,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}

// Breadcrumb is a suggested follow-up action.
type Breadcrumb struct {
	Action      string `json:"action"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

// ErrorResponse is the error envelope for JSON output.
type ErrorResponse struct {
	OK    bool           `json:"ok"`
	Error string         `json:"error"`
	Code  string         `json:"code"`
	Hint  string         `json:"hint,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
}

// Format specifies the output format.
type Format int

const (
	FormatAuto     Format = iota // Auto-detect: TTY -> Styled, non-TTY -> JSON
	FormatJSON                   // JSON envelope
	FormatMarkdown               // Literal Markdown syntax
	FormatStyled                 // ANSI styled output
	FormatQuiet                  // Raw JSON data, no envelope
	FormatIDs                    // One ID per line
	FormatCount                  // Integer count only
)

// Options controls output behavior.
type Options struct {
	Format  Format
	Writer  io.Writer
	Verbose bool
}

// DefaultOptions returns options for standard output.
func DefaultOptions() Options {
	return Options{
		Format: FormatAuto,
		Writer: os.Stdout,
	}
}

// Writer handles all output formatting.
type Writer struct {
	opts Options
}

// New creates a new output writer.
func New(opts Options) *Writer {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	return &Writer{opts: opts}
}

// EffectiveFormat resolves FormatAuto based on TTY detection.
func (w *Writer) EffectiveFormat() Format {
	if w.opts.Format == FormatAuto {
		if isTTY(w.opts.Writer) {
			return FormatStyled
		}
		return FormatJSON
	}
	return w.opts.Format
}

// OK outputs a success response.
func (w *Writer) OK(data any, opts ...ResponseOption) error {
	resp := &Response{OK: true, Data: data}
	for _, opt := range opts {
		opt(resp)
	}
	return w.write(resp)
}

// Err outputs an error response.
func (w *Writer) Err(err error, opts ...ErrorResponseOption) error {
	e := AsError(err)
	resp := &ErrorResponse{
		OK:    false,
		Error: e.Message,
		Code:  e.Code,
		Hint:  e.Hint,
	}
	for _, opt := range opts {
		opt(resp)
	}
	return w.write(resp)
}

// ResponseOption modifies a Response.
type ResponseOption func(*Response)

// ErrorResponseOption modifies an ErrorResponse.
type ErrorResponseOption func(*ErrorResponse)

// WithSummary adds a summary to the response.
func WithSummary(s string) ResponseOption {
	return func(r *Response) { r.Summary = s }
}

// WithNotice adds an informational notice to the response.
func WithNotice(s string) ResponseOption {
	return func(r *Response) { r.Notice = s }
}

// TruncationNotice returns a notice string if results may be truncated.
// Returns empty string if no truncation warning is needed.
func TruncationNotice(count, defaultLimit int, all bool, explicitLimit int) string {
	if all {
		return ""
	}

	limit := defaultLimit
	if explicitLimit > 0 {
		limit = explicitLimit
	}

	if limit == 0 {
		return ""
	}

	if count > 0 && count >= limit {
		return fmt.Sprintf("Showing %d results (use --all for complete list)", count)
	}

	return ""
}

// TruncationNoticeWithTotal returns a truncation notice when results are truncated.
// Uses totalCount from API's X-Total-Count header to show accurate counts.
// Returns empty string if no truncation or totalCount is 0 (unavailable).
func TruncationNoticeWithTotal(count, totalCount int) string {
	if totalCount == 0 || count >= totalCount {
		return ""
	}

	return fmt.Sprintf("Showing %d of %d results (use --all for complete list)", count, totalCount)
}

// WithBreadcrumbs adds breadcrumbs to the response.
func WithBreadcrumbs(b ...Breadcrumb) ResponseOption {
	return func(r *Response) { r.Breadcrumbs = append(r.Breadcrumbs, b...) }
}

// WithoutBreadcrumbs removes all breadcrumbs from the response.
func WithoutBreadcrumbs() ResponseOption {
	return func(r *Response) { r.Breadcrumbs = nil }
}

// WithContext adds context to the response.
func WithContext(key string, value any) ResponseOption {
	return func(r *Response) {
		if r.Context == nil {
			r.Context = make(map[string]any)
		}
		r.Context[key] = value
	}
}

// WithMeta adds metadata to the response.
func WithMeta(key string, value any) ResponseOption {
	return func(r *Response) {
		if r.Meta == nil {
			r.Meta = make(map[string]any)
		}
		r.Meta[key] = value
	}
}

func (w *Writer) write(v any) error {
	format := w.opts.Format

	if format == FormatAuto {
		if isTTY(w.opts.Writer) {
			format = FormatStyled
		} else {
			format = FormatJSON
		}
	}

	switch format {
	case FormatQuiet:
		if resp, ok := v.(*Response); ok {
			return w.writeQuiet(resp.Data)
		}
		return w.writeQuiet(v)
	case FormatIDs:
		return w.writeIDs(v)
	case FormatCount:
		return w.writeCount(v)
	case FormatStyled, FormatMarkdown:
		// The shared package doesn't do rendering -- apps provide that.
		// Fall back to JSON.
		return w.writeJSON(v)
	default:
		return w.writeJSON(v)
	}
}

// isTTY checks if the writer is a terminal.
func isTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func (w *Writer) writeJSON(v any) error {
	toEncode := v
	if resp, ok := v.(*Response); ok {
		respCopy := *resp
		respCopy.Data = NormalizeData(resp.Data)
		toEncode = &respCopy
	}
	enc := json.NewEncoder(w.opts.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(toEncode)
}

func (w *Writer) writeQuiet(v any) error {
	return w.writeJSON(NormalizeData(v))
}

func (w *Writer) writeIDs(v any) error {
	resp, ok := v.(*Response)
	if !ok {
		return w.writeJSON(v)
	}

	data := NormalizeData(resp.Data)

	switch d := data.(type) {
	case []map[string]any:
		for _, item := range d {
			if id, ok := item["id"]; ok {
				if _, err := fmt.Fprintln(w.opts.Writer, id); err != nil {
					return err
				}
			}
		}
	case map[string]any:
		if id, ok := d["id"]; ok {
			if _, err := fmt.Fprintln(w.opts.Writer, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Writer) writeCount(v any) error {
	resp, ok := v.(*Response)
	if !ok {
		return w.writeJSON(v)
	}

	data := NormalizeData(resp.Data)

	switch d := data.(type) {
	case nil:
		_, err := fmt.Fprintln(w.opts.Writer, 0)
		return err
	case []any:
		_, err := fmt.Fprintln(w.opts.Writer, len(d))
		return err
	case []map[string]any:
		_, err := fmt.Fprintln(w.opts.Writer, len(d))
		return err
	default:
		_, err := fmt.Fprintln(w.opts.Writer, 1)
		return err
	}
}

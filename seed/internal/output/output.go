// Package output wraps github.com/basecamp/cli/output with app-specific additions.
//
// This is a seed template. Customize it for your CLI.
package output

import (
	"github.com/basecamp/cli/output"
)

// Re-export core types for convenience.
type (
	Response       = output.Response
	ErrorResponse  = output.ErrorResponse
	Breadcrumb     = output.Breadcrumb
	Format         = output.Format
	Options        = output.Options
	Writer         = output.Writer
	Error          = output.Error
	ResponseOption = output.ResponseOption
)

// Re-export format constants.
const (
	FormatAuto     = output.FormatAuto
	FormatJSON     = output.FormatJSON
	FormatMarkdown = output.FormatMarkdown
	FormatStyled   = output.FormatStyled
	FormatQuiet    = output.FormatQuiet
	FormatIDs      = output.FormatIDs
	FormatCount    = output.FormatCount
)

// Re-export constructors and helpers.
var (
	New             = output.New
	DefaultOptions  = output.DefaultOptions
	WithSummary     = output.WithSummary
	WithNotice      = output.WithNotice
	WithBreadcrumbs = output.WithBreadcrumbs
	WithContext     = output.WithContext
	WithMeta        = output.WithMeta
)

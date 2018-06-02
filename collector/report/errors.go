package report

import "errors"

// ErrReportIsClosed indicates the report has already been closed
var ErrReportIsClosed = errors.New("Report is already closed")

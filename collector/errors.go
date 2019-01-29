package collector

import "errors"

// ErrReportIsClosed indicates the report has already been closed
var ErrReportIsClosed = errors.New("Report is already closed")

// ErrReportNotFound indicates no report with the given id could be found
var ErrReportNotFound = errors.New("Report not found")

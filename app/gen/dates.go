package gen

import "time"

// Date layouts: the ISO form used in site.json and metadata, the older month-and-year form, and the form shown to
// readers.
const (
	isoDateLayout     = "2006-01-02"
	monthYearLayout   = "January 2006"
	displayDateLayout = "January 2, 2006"
)

// displayDate is a date as shown under an article's title: a full date reads "August 21, 2026", anything else as
// written.
func displayDate(date string) string {
	if t, err := time.Parse(isoDateLayout, date); err == nil {
		return t.Format(displayDateLayout)
	}
	return date
}

// isoDate is a date as YYYY-MM-DD, "" when it is empty or in neither of the layouts site.json accepts.
func isoDate(date string) string {
	if date == "" {
		return ""
	}
	if _, err := time.Parse(isoDateLayout, date); err == nil {
		return date
	}
	if t, err := time.Parse(monthYearLayout, date); err == nil {
		return t.Format(isoDateLayout)
	}
	return ""
}

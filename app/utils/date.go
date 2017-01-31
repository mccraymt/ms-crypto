package utils

import (
	"strconv"
	"time"
)

// ConstDateFormatWeb => web date format constant
const ConstDateFormatWeb = "01-02-2006"

// ConstDateFormatPC => pc date format constant
const ConstDateFormatPC = "2006/01/02"

// ConstTimestampFormatPC => pc timestamp format
const ConstTimestampFormatPC = "2006-01-02 15:04:05 -0700"

// WebDateToPCDate => converts a web date (MM-DD-YYYY) to a pc date (YYYY/MM/DD)
func WebDateToPCDate(webDate string) (string, error) {
	var rtn time.Time
	var err error

	if rtn, err = time.Parse(ConstDateFormatWeb, webDate); err != nil {
		return "", err
	}

	return TimeToPCDate(&rtn), nil
}

// PCDateToWebDate => converts a pc date (YYYY/MM/DD) to a web date (MM-DD-YYYY)
func PCDateToWebDate(pcDate string) (string, error) {
	var rtn time.Time
	var err error

	if rtn, err = time.Parse(ConstDateFormatPC, pcDate); err != nil {
		return "", err
	}

	return TimeToWebDate(&rtn), nil
}

// WebDateToPCTimestamp => converts a web date (MM-DD-YYYY) to a pc timestamp (YYYY-MM-DD hh:mm:ss -0700)
func WebDateToPCTimestamp(webDate string) (string, error) {
	var rtn time.Time
	var err error

	if rtn, err = time.Parse(ConstDateFormatWeb, webDate); err != nil {
		return "", err
	}

	return rtn.Format(ConstTimestampFormatPC), nil
}

// PCTimestampToWebDate => converts a pc timestamp (YYYY-MM-DD hh:mm:ss -0700) to web date (MM-DD-YYYY)
func PCTimestampToWebDate(pcDate string, asUTC bool) (string, error) {
	var rtn time.Time
	var err error

	if rtn, err = time.Parse(ConstTimestampFormatPC, pcDate); err != nil {
		return "", err
	}

	// return as UTC date?
	if asUTC {
		rtnUtc := rtn.UTC()
		return TimeToWebDate(&rtnUtc), nil
	}

	// return as non-UTC date
	return TimeToWebDate(&rtn), nil
}

// DateStringToYearMonth => parsed year (int) and month (int) from Date (string)
func DateStringToYearMonth(date string) (int, int) {
	year := 0
	month := 0
	if date != "" {
		if dt, fail := StrconvToDate(date); fail == nil {
			year = dt.Year()
			month = int(dt.Month())
		}
	}
	return year, month
}

// ParsePCTimestamp returns a time from a PC timestamp string
func ParsePCTimestamp(ts string) *time.Time {
	if rtn, err := time.Parse(ConstTimestampFormatPC, ts); err == nil {
		return PtrToTime(rtn.UTC())
	}
	return nil
}

// PtrToTime => returns a pointer to the provided time
func PtrToTime(t time.Time) *time.Time {
	return &t
}

// TimeToPCDate returns a PC date string from a time
func TimeToPCDate(t *time.Time) string {
	if t != nil {
		return t.Format(ConstDateFormatPC)
	}
	return ""
}

// TimeToWebDate returns a web date string from a time
func TimeToWebDate(t *time.Time) string {
	if t != nil {
		return t.Format(ConstDateFormatWeb)
	}
	return ""
}

// WebDateFromYearMonthStrings does just what you think it does
func WebDateFromYearMonthStrings(year, month string) string {
	y, yerr := strconv.Atoi(year)
	if yerr != nil || y < 1800 || y > 2200 {
		return ""
	}

	m, _ := strconv.Atoi(month)
	if m < 1 || m > 31 {
		m = 1
	}

	rtnTime := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)

	return TimeToWebDate(&rtnTime)
}

package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var pcPhoneRegex = regexp.MustCompile("^([2-9][0-9]{2})-([2-9][0-9]{2})-([0-9]{4})$")
var webPhoneRegex = regexp.MustCompile("^([2-9][0-9]{2})([2-9][0-9]{2})([0-9]{4})$")

// StrconvWebPhoneToPCPhone => convert web phone format to PC phone format
func StrconvWebPhoneToPCPhone(webPhone string) (string, error) {
	if pcPhoneRegex.MatchString(webPhone) {
		return webPhone, nil
	}

	if !webPhoneRegex.MatchString(webPhone) {
		return "", fmt.Errorf("Phone \"%v\" not recognized as a valid web phone number", webPhone)
	}

	pt := webPhoneRegex.FindStringSubmatch(webPhone)

	return fmt.Sprintf("%v-%v-%v", pt[1], pt[2], pt[3]), nil
}

// StrconvWebDateOfBirthToPCDateOfBirth => convert web birth date format to PC birth date format
// EXAMPLE of Web birth date: "12-30-1980"
// EXAMPLE of PC birth date: "1980/12/30"
func StrconvWebDateOfBirthToPCDateOfBirth(webDateOfBirth string) (string, error) {
	if webDateOfBirth == "" {
		return "", errors.New("StrconvWebDateOfBirthToPCDateOfBirth: birth date cannot be blank")
	}

	arrDDMMYYYY := strings.Split(webDateOfBirth, "-")

	if len(arrDDMMYYYY) != 3 {
		return "", errors.New("StrconvWebDateOfBirthToPCDateOfBirth: birth date is improperly formatted. Should be DD-MM-YYYY")
	}

	day := arrDDMMYYYY[0]
	month := arrDDMMYYYY[1]
	year := arrDDMMYYYY[2]

	if len(day) != 2 {
		return "", errors.New("StrconvWebDateOfBirthToPCDateOfBirth: birth day must be 2 digits")
	} else if len(month) != 2 {
		return "", errors.New("StrconvWebDateOfBirthToPCDateOfBirth: birth month must be 2 digits")
	} else if len(year) != 4 {
		return "", errors.New("StrconvWebDateOfBirthToPCDateOfBirth: birth date year must be four digits")
	}

	return fmt.Sprintf("%s/%s/%s", year, month, day), nil
}

// StrconvToDate converts yyyy/MM/dd or yyyy-MM-dd strings to Times
func StrconvToDate(strDate string) (time.Time, error) {
	//_ = "breakpoint"
	//fmt.Println("StrconvToDate", strDate)
	if strDate != "" {
		datePattern := ""
		idxs := strings.IndexRune(strDate, '/')
		idxd := strings.IndexRune(strDate, '-')
		switch {
		case idxs == 2 && idxd < 0:
			datePattern = "01/02/2006"
			break
		case idxs == 4 && idxd < 0:
			datePattern = "2006/01/02"
			break
		case idxd == 2 && idxs < 0:
			datePattern = "01-02-2006"
			break
		case idxd == 4 && idxs < 0:
			datePattern = "2006-01-02"
		}

		if len(datePattern) == 0 {
			return time.Now(), fmt.Errorf("Date string %v not recognized as a valid date", strDate)
		}

		t, err := time.ParseInLocation(datePattern, strDate, time.UTC)
		if err != nil {
			panic(err)
		}
		return t, err
	}
	return time.Now(), errors.New("There is no date string to convert")
}

// FormatQuoteRetrieveKey => builds the quote retrieve key based on lastName, dob, and email
func FormatQuoteRetrieveKey(lastName string, dob string, email string) string {
	if lastName == "" || dob == "" || email == "" {
		return ""
	}

	return fmt.Sprintf("%v|%v|%v", strings.ToLower(lastName), dob, strings.ToLower(email))
}

// GenerateQuoteNumber => generates a reasonably random quote number
func GenerateQuoteNumber() string {
	return QuoteNumber{}.CreateQuoteNumber(nil)
}

// PtrToStr => returns a pointer to the provided string
func PtrToStr(s string) *string {
	return &s
}

// PtrToStrCopy => returns a pointer to a copy of a string, for strings whose addresses can't be taken
func PtrToStrCopy(s string) *string {
	c := s
	return &c
}

// DerefStr => derenferences a string if it's not nil, otherwise returns ""
func DerefStr(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

// StrEqual => compare two strings with an option for case insensitive comparisons
func StrEqual(s1 string, s2 string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.ToLower(s1) == strings.ToLower(s2)
	}

	return s1 == s2
}

// ToLowerCompare returns true if two strings are identical apart from case
func ToLowerCompare(str1, str2 string) bool {
	return strings.ToLower(str1) == strings.ToLower(str2)
}

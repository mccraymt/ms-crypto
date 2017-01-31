// Package utils => age allows for easy calculation of the age of an entity, provided with the date of birth of that entity.
package utils

import "time"

// AgeAt gets the age of an entity at a certain time.
func AgeAt(DateOfBirth time.Time, now time.Time) int {
	// Get the year number change since the player's birth.
	years := now.Year() - DateOfBirth.Year()

	// If the date is before the date of birth, then not that many years have elapsed.
	birthDay := getAdjustedBirthDay(DateOfBirth, now)
	if now.YearDay() < birthDay {
		years--
	}

	return years
}

// Age is shorthand for AgeAt(DateOfBirth, time.Now()), and carries the same usage and limitations.
func Age(DateOfBirth time.Time) int {
	return AgeAt(DateOfBirth, time.Now())
}

// Gets the adjusted date of birth to work around leap year differences.
func getAdjustedBirthDay(DateOfBirth time.Time, now time.Time) int {
	birthDay := DateOfBirth.YearDay()
	currentDay := now.YearDay()
	if isLeap(DateOfBirth) && !isLeap(now) && birthDay >= 60 {
		return birthDay - 1
	}
	if isLeap(now) && !isLeap(DateOfBirth) && currentDay >= 60 {
		return birthDay + 1
	}
	return birthDay
}

// Works out if a time.Time is in a leap year.
func isLeap(date time.Time) bool {
	year := date.Year()
	if year%400 == 0 {
		return true
	} else if year%100 == 0 {
		return false
	} else if year%4 == 0 {
		return true
	}
	return false
}

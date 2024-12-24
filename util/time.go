package util

import "time"

/*
CreateTimestampStr Create a new timestamp representing the current date time and return it as
a string
*/
func CreateTimestampStr() string {
	currentDate := time.Now()
	return currentDate.String()
}

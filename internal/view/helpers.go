package view

import "time"

const sqliteTimestamp = "2006-01-02 15:04:05"

// FormatLocalTime turns a SQLite UTC timestamp into HH:MM in local time.
func FormatLocalTime(s string) string {
	t, err := time.Parse(sqliteTimestamp, s)
	if err != nil {
		return s
	}
	return t.Local().Format("15:04")
}

// FormatLocalDate turns a SQLite UTC timestamp into YYYY-MM-DD in local time.
func FormatLocalDate(s string) string {
	t, err := time.Parse(sqliteTimestamp, s)
	if err != nil {
		return s
	}
	return t.Local().Format("2006-01-02")
}

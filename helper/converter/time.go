package helperconverter

import (
	"fmt"
	"strings"
	"time"
)

func ConvertTimeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	layout := "2006-01-02 15:04:05"
	return t.Format(layout)
}

func ConvertStringToDate(tString string) (time.Time, error) {
	layout := "2006-01-02"

	t, err := time.Parse(layout, tString)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

var layouts = []string{
	"1/2/2006 3:04:05 PM",
	"1/2/2006 3:04 PM",
	"1/2/2006 15:04:05",
	"1/2/2006 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
}

func ConvertStringToTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported datetime format: %s", s)
}

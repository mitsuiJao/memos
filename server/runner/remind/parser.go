package remind

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RemindCommand holds the parsed result of a #remind memo.
// Exactly one of IsSchedule, IsListCommand, or IsDelCommand is true unless ParseError is set.
// A nil return from ParseRemindContent means the memo has no #remind tag.
type RemindCommand struct {
	IsSchedule    bool
	ScheduledTime time.Time
	Description   string

	IsListCommand bool

	IsDelCommand   bool
	DisplayNumber  int // 1-based display number from "#remind del N"

	ParseError error
}

// ParseRemindContent parses memo content for a #remind command.
// Returns nil if no #remind tag is found at the start or end of content.
// Returns a non-nil RemindCommand (possibly with ParseError) when #remind is found.
func ParseRemindContent(content string, now time.Time) *RemindCommand {
	content = strings.TrimSpace(content)

	var rest string
	switch {
	case strings.HasPrefix(content, "#remind"):
		rest = strings.TrimSpace(content[len("#remind"):])
	case strings.HasSuffix(content, "#remind"):
		rest = strings.TrimSpace(content[:len(content)-len("#remind")])
	default:
		return nil
	}

	// #remind list
	if strings.EqualFold(rest, "list") {
		return &RemindCommand{IsListCommand: true}
	}

	// #remind del N
	if lower := strings.ToLower(rest); strings.HasPrefix(lower, "del ") || lower == "del" {
		arg := strings.TrimSpace(rest[3:])
		n, err := strconv.Atoi(arg)
		if err != nil || n < 1 {
			return &RemindCommand{ParseError: fmt.Errorf("削除番号が不正です: %q — 例: `#remind del 1`", arg)}
		}
		return &RemindCommand{IsDelCommand: true, DisplayNumber: n}
	}

	// #remind <time> [description]
	tokens := strings.Fields(rest)
	if len(tokens) == 0 {
		return &RemindCommand{ParseError: fmt.Errorf("時刻が指定されていません — 例: `#remind 1130 洗濯`")}
	}

	timeStr := tokens[0]
	description := strings.Join(tokens[1:], " ")

	t, err := parseTime(timeStr, now)
	if err != nil {
		return &RemindCommand{ParseError: err}
	}

	return &RemindCommand{
		IsSchedule:    true,
		ScheduledTime: t,
		Description:   description,
	}
}

// parseTime converts a 4-digit (hhmm) or 8-digit (mmddhhmm) string to a time.Time in UTC.
func parseTime(s string, now time.Time) (time.Time, error) {
	if !isAllDigits(s) {
		return time.Time{}, fmt.Errorf("時刻フォーマットが不正です: %q — 数字のみ使用可 (例: 1130 または 05141130)", s)
	}

	switch len(s) {
	case 4:
		hour, _ := strconv.Atoi(s[0:2])
		min, _ := strconv.Atoi(s[2:4])
		if hour > 23 || min > 59 {
			return time.Time{}, fmt.Errorf("時刻が範囲外です: %q (hh=00-23, mm=00-59)", s)
		}
		t := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.UTC)
		if !t.After(now) {
			t = t.Add(24 * time.Hour)
		}
		return t, nil

	case 8:
		month, _ := strconv.Atoi(s[0:2])
		day, _ := strconv.Atoi(s[2:4])
		hour, _ := strconv.Atoi(s[4:6])
		min, _ := strconv.Atoi(s[6:8])
		if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || min > 59 {
			return time.Time{}, fmt.Errorf("日時が範囲外です: %q (mm=01-12, dd=01-31, hh=00-23, mm=00-59)", s)
		}
		t := time.Date(now.Year(), time.Month(month), day, hour, min, 0, 0, time.UTC)
		if !t.After(now) {
			t = t.AddDate(1, 0, 0)
		}
		return t, nil

	default:
		return time.Time{}, fmt.Errorf("時刻フォーマットが不正です: %q — 4桁(hhmm)または8桁(mmddhhmm)で指定してください", s)
	}
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

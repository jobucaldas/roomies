package recurrence

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

const (
	MaxInterval   = 366
	MaxCount      = 366
	MaxExceptions = 100
	MaxOutputs    = 366
)

type Spec struct {
	Timezone     string
	DTStartLocal string
	Rule         string
	ExDates      []string
}

func Validate(spec Spec, now time.Time) error {
	loc, err := time.LoadLocation(spec.Timezone)
	if err != nil {
		return errors.New("timezone must be a valid IANA timezone")
	}
	start, err := time.ParseInLocation("2006-01-02T15:04:05", spec.DTStartLocal, loc)
	if err != nil {
		return errors.New("dtstart_local must use YYYY-MM-DDTHH:MM:SS")
	}
	if len(spec.ExDates) > MaxExceptions {
		return fmt.Errorf("at most %d exception dates are allowed", MaxExceptions)
	}
	parts := map[string]string{}
	for _, raw := range strings.Split(strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(spec.Rule)), "RRULE:"), ";") {
		pair := strings.SplitN(raw, "=", 2)
		if len(pair) != 2 || pair[0] == "" || pair[1] == "" {
			return errors.New("invalid RRULE")
		}
		parts[pair[0]] = pair[1]
	}
	if parts["FREQ"] != "DAILY" && parts["FREQ"] != "WEEKLY" && parts["FREQ"] != "MONTHLY" {
		return errors.New("RRULE FREQ must be DAILY, WEEKLY, or MONTHLY")
	}
	if value := parts["INTERVAL"]; value != "" {
		n, e := strconv.Atoi(value)
		if e != nil || n < 1 || n > MaxInterval {
			return fmt.Errorf("RRULE INTERVAL must be between 1 and %d", MaxInterval)
		}
	}
	if value := parts["COUNT"]; value != "" {
		n, e := strconv.Atoi(value)
		if e != nil || n < 1 || n > MaxCount {
			return fmt.Errorf("RRULE COUNT must be between 1 and %d", MaxCount)
		}
	}
	if parts["COUNT"] == "" && parts["UNTIL"] == "" {
		return errors.New("RRULE must contain bounded COUNT or UNTIL")
	}
	if value := parts["UNTIL"]; value != "" {
		until, e := parseUntil(value, loc)
		if e != nil {
			return errors.New("invalid RRULE UNTIL")
		}
		if until.After(now.AddDate(1, 0, 0)) || until.After(start.AddDate(1, 0, 0)) {
			return errors.New("RRULE UNTIL must be within one year")
		}
	}
	allowed := map[string]bool{"FREQ": true, "INTERVAL": true, "COUNT": true, "UNTIL": true, "BYDAY": true, "BYMONTHDAY": true, "BYMONTH": true, "WKST": true}
	for key := range parts {
		if !allowed[key] {
			return fmt.Errorf("unsupported RRULE field %s", key)
		}
	}
	if _, err := build(spec); err != nil {
		return fmt.Errorf("invalid recurrence: %w", err)
	}
	return nil
}

func Expand(spec Spec, after, before time.Time, limit int) ([]time.Time, error) {
	if limit <= 0 || limit > MaxOutputs {
		limit = MaxOutputs
	}
	set, err := build(spec)
	if err != nil {
		return nil, err
	}
	values := set.Between(after, before, false)
	if len(values) > limit {
		values = values[:limit]
	}
	result := make([]time.Time, len(values))
	for i, value := range values {
		result[i] = value.UTC()
	}
	return result, nil
}

func Next(spec Spec, after time.Time) (*time.Time, error) {
	set, err := build(spec)
	if err != nil {
		return nil, err
	}
	value := set.After(after, false)
	if value.IsZero() {
		return nil, nil
	}
	value = value.UTC()
	return &value, nil
}

func build(spec Spec) (*rrule.Set, error) {
	loc, err := time.LoadLocation(spec.Timezone)
	if err != nil {
		return nil, err
	}
	start, err := time.ParseInLocation("2006-01-02T15:04:05", spec.DTStartLocal, loc)
	if err != nil {
		return nil, err
	}
	rule, err := rrule.StrToRRule(strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(spec.Rule)), "RRULE:"))
	if err != nil {
		return nil, err
	}
	rule.DTStart(start)
	set := &rrule.Set{}
	set.RRule(rule)
	for _, raw := range spec.ExDates {
		value, e := time.ParseInLocation("2006-01-02T15:04:05", raw, loc)
		if e != nil {
			return nil, fmt.Errorf("invalid EXDATE %q", raw)
		}
		set.ExDate(value)
	}
	return set, nil
}

func parseUntil(value string, loc *time.Location) (time.Time, error) {
	for _, format := range []string{"20060102T150405Z", "20060102T150405", "20060102"} {
		if strings.HasSuffix(value, "Z") {
			if t, e := time.Parse(format, value); e == nil {
				return t, nil
			}
		} else if t, e := time.ParseInLocation(format, value, loc); e == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid UNTIL")
}

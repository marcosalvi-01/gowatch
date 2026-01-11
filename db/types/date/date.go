package date

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

const Layout = "2006-01-02"

type Date struct {
	time.Time
}

func New(t time.Time) Date {
	return Date{Time: t}
}

func NewFromPtr(t *time.Time) Date {
	if t == nil {
		return Date{}
	}
	return Date{Time: *t}
}

func (d Date) ToTimePtr() *time.Time {
	if d.Time.IsZero() {
		return nil
	}
	t := d.Time
	return &t
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		d.Time = time.Time{}
		return nil
	}
	t, err := time.Parse(Layout, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", d.Time.Format(Layout))), nil
}

func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		d.Time = v
		return nil
	case []byte:
		return d.parse(string(v))
	case string:
		return d.parse(v)
	default:
		return fmt.Errorf("cannot scan type %T into Date", value)
	}
}

func (d *Date) parse(s string) error {
	// Try standard layout
	t, err := time.Parse(Layout, s)
	if err == nil {
		d.Time = t
		return nil
	}

	// Fallback to RFC3339 if needed (sometimes drivers return full timestamp)
	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		d.Time = t
		return nil
	}

	// Try space separated
	t, err = time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		d.Time = t
		return nil
	}

	return fmt.Errorf("cannot parse date string %q", s)
}

func (d Date) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time.Format(Layout), nil
}

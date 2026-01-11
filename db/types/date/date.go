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
	if d.IsZero() {
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
	if d.IsZero() {
		return []byte("null"), nil
	}
	return fmt.Appendf(nil, "\"%s\"", d.Format(Layout)), nil
}

func (d *Date) Scan(value any) error {
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
	t, err := time.Parse(Layout, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", s)
			if err != nil {
				return fmt.Errorf("cannot parse date string %q", s)
			}
		}
	}

	d.Time = t
	return nil
}

func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.Format(Layout), nil
}

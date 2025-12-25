package cli

import (
	"strconv"
	"time"
)

// OptionalDuration records a duration flag and whether it was set.
type OptionalDuration struct {
	value time.Duration
	set   bool
}

func (o *OptionalDuration) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

func (o *OptionalDuration) String() string {
	if !o.set {
		return ""
	}
	return o.value.String()
}

func (o *OptionalDuration) Value() (time.Duration, bool) {
	return o.value, o.set
}

// OptionalInt records an int flag and whether it was set.
type OptionalInt struct {
	value int
	set   bool
}

func (o *OptionalInt) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

func (o *OptionalInt) String() string {
	if !o.set {
		return ""
	}
	return strconv.Itoa(o.value)
}

func (o *OptionalInt) Value() (int, bool) {
	return o.value, o.set
}

// OptionalString records a string flag and whether it was set.
type OptionalString struct {
	value string
	set   bool
}

func (o *OptionalString) Set(s string) error {
	o.value = s
	o.set = true
	return nil
}

func (o *OptionalString) String() string {
	if !o.set {
		return ""
	}
	return o.value
}

func (o *OptionalString) Value() (string, bool) {
	return o.value, o.set
}

// OptionalBool records a bool flag and whether it was set.
type OptionalBool struct {
	value bool
	set   bool
}

func (o *OptionalBool) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

func (o *OptionalBool) String() string {
	if !o.set {
		return ""
	}
	if o.value {
		return "true"
	}
	return "false"
}

func (o *OptionalBool) IsBoolFlag() bool {
	return true
}

func (o *OptionalBool) Value() (bool, bool) {
	return o.value, o.set
}

package bannerlord

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

const maxVersionPart = 1<<31 - 1

// Version is a Bannerlord application version such as "v1.2.3.0" or "e1.0.0": a channel prefix
// letter (a alpha, b beta, e early access, v release, d development) and three or four
// dot-separated numbers. The zero Version means "unspecified".
type Version struct {
	prefix byte
	parts  [4]int
	n      int
}

// channels lists the prefix letters in ApplicationVersionType order.
const channels = "abevd"

// ParseVersion parses a version string as ApplicationVersion.TryParse does: the prefix letter is
// required, there are three or four parts, and a "*" part (as in "v1.2.*") and every part after it
// count as zero, which is the minimum the wildcard matches. It refuses negative or oversized
// numbers and unknown prefixes.
func ParseVersion(s string) (Version, error) {
	var v Version
	if s == "" || !strings.ContainsRune(channels, rune(s[0])) {
		return Version{}, fmt.Errorf("%w: version %q", ErrInvalid, s)
	}
	v.prefix = s[0]
	fields := strings.Split(s[1:], ".")
	if len(fields) < 3 || len(fields) > len(v.parts) {
		return Version{}, fmt.Errorf("%w: version %q needs three or four parts", ErrInvalid, s)
	}
	for i, f := range fields {
		if f == "*" {
			break
		}
		num, err := strconv.Atoi(f)
		if err != nil || num < 0 || num > maxVersionPart || f != strings.TrimSpace(f) || strings.HasPrefix(f, "+") || strings.HasPrefix(f, "-") {
			return Version{}, fmt.Errorf("%w: version %q", ErrInvalid, s)
		}
		v.parts[i] = num
	}
	v.n = len(fields)
	return v, nil
}

// IsZero reports whether the version is unspecified.
func (v Version) IsZero() bool { return v.n == 0 }

// Compare returns -1, 0 or 1 as v is older than, equal to or newer than o. The channel prefix
// compares first (a < b < e < v < d), then the numbers, as ApplicationVersionComparer does.
func (v Version) Compare(o Version) int {
	if r, or := strings.IndexByte(channels, v.prefix), strings.IndexByte(channels, o.prefix); r != or {
		return cmp.Compare(r, or)
	}
	for i := range v.parts {
		switch {
		case v.parts[i] < o.parts[i]:
			return -1
		case v.parts[i] > o.parts[i]:
			return 1
		}
	}
	return 0
}

// String renders the prefix and the parsed parts; a wildcard renders as zero.
func (v Version) String() string {
	if v.n == 0 {
		return ""
	}
	var b strings.Builder
	if v.prefix != 0 {
		b.WriteByte(v.prefix)
	}
	for i := range v.n {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.Itoa(v.parts[i]))
	}
	return b.String()
}

package json

import (
	"regexp"

	"github.com/charlienet/go-misc/bytesconv"
	"github.com/charlienet/go-misc/stringx"
)

var (
	keyMatchRegex = regexp.MustCompile(`\"(\w+)\":`)
)

type Snake2Camel struct{ Value any }

func (c Snake2Camel) MarshalJSON() ([]byte, error) {
	marshalled, err := Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := bytesconv.BytesToString(match[1 : len(match)-2])
			name := stringx.Snake2Camel(matchStr)
			return []byte(`"` + name + `":`)
		})

	return converted, err
}

type Snake2Pascal struct{ Value any }

func (c Snake2Pascal) MarshalJSON() ([]byte, error) {
	marshalled, err := Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := bytesconv.BytesToString(match[1 : len(match)-2])
			name := stringx.Snake2Pascal(matchStr)
			return []byte(`"` + name + `":`)
		})

	return converted, err
}

type Pascal2Camel struct{ Value any }

func (c Pascal2Camel) MarshalJSON() ([]byte, error) {
	marshalled, err := Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := bytesconv.BytesToString(match[1 : len(match)-2])
			name := stringx.Pascal2Camel(matchStr)
			return []byte(`"` + name + `":`)
		})

	return converted, err
}

type Camel2Pascal struct{ Value any }

func (c Camel2Pascal) MarshalJSON() ([]byte, error) {
	marshalled, err := Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := bytesconv.BytesToString(match[1 : len(match)-2])
			name := stringx.Camel2Pascal(matchStr)
			return []byte(`"` + name + `":`)
		})

	return converted, err
}

type Pascal2Snake struct{ Value any }

func (c Pascal2Snake) MarshalJSON() ([]byte, error) {
	marshalled, err := Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := bytesconv.BytesToString(match[1 : len(match)-2])
			name := stringx.Pascal2Snake(matchStr)
			return []byte(`"` + name + `":`)
		})

	return converted, err
}

type Pascal2UpperSnake struct{ Value any }

func (c Pascal2UpperSnake) MarshalJSON() ([]byte, error) {
	marshalled, err := Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := bytesconv.BytesToString(match[1 : len(match)-2])
			name := stringx.Pascal2UpperSnake(matchStr)
			return []byte(`"` + name + `":`)
		})

	return converted, err
}

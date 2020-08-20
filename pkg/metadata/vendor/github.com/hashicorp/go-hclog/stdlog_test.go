package hclog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdlogAdapter(t *testing.T) {
	t.Run("picks debug level", func(t *testing.T) {
		var s stdlogAdapter

		level, rest := s.pickLevel("[DEBUG] coffee?")

		assert.Equal(t, Debug, level)
		assert.Equal(t, "coffee?", rest)
	})

	t.Run("picks trace level", func(t *testing.T) {
		var s stdlogAdapter

		level, rest := s.pickLevel("[TRACE] coffee?")

		assert.Equal(t, Trace, level)
		assert.Equal(t, "coffee?", rest)
	})

	t.Run("picks info level", func(t *testing.T) {
		var s stdlogAdapter

		level, rest := s.pickLevel("[INFO] coffee?")

		assert.Equal(t, Info, level)
		assert.Equal(t, "coffee?", rest)
	})

	t.Run("picks warn level", func(t *testing.T) {
		var s stdlogAdapter

		level, rest := s.pickLevel("[WARN] coffee?")

		assert.Equal(t, Warn, level)
		assert.Equal(t, "coffee?", rest)
	})

	t.Run("picks error level", func(t *testing.T) {
		var s stdlogAdapter

		level, rest := s.pickLevel("[ERROR] coffee?")

		assert.Equal(t, Error, level)
		assert.Equal(t, "coffee?", rest)
	})

	t.Run("picks error as err level", func(t *testing.T) {
		var s stdlogAdapter

		level, rest := s.pickLevel("[ERR] coffee?")

		assert.Equal(t, Error, level)
		assert.Equal(t, "coffee?", rest)
	})
}

func TestStdlogAdapter_ForceLevel(t *testing.T) {
	cases := []struct {
		name        string
		forceLevel  Level
		inferLevels bool
		write       string
		expect      string
	}{
		{
			name:       "force error",
			forceLevel: Error,
			write:      "this is a test",
			expect:     "[ERROR] test: this is a test\n",
		},
		{
			name:        "force error overrides infer",
			forceLevel:  Error,
			inferLevels: true,
			write:       "[DEBUG] this is a test",
			expect:      "[ERROR] test: this is a test\n",
		},
		{
			name:       "force error and strip debug",
			forceLevel: Error,
			write:      "[DEBUG] this is a test",
			expect:     "[ERROR] test: this is a test\n",
		},
		{
			name:       "force trace",
			forceLevel: Trace,
			write:      "this is a test",
			expect:     "[TRACE] test: this is a test\n",
		},
		{
			name:       "force trace and strip higher level error",
			forceLevel: Trace,
			write:      "[WARN] this is a test",
			expect:     "[TRACE] test: this is a test\n",
		},
		{
			name:       "force with invalid level",
			forceLevel: -10,
			write:      "this is a test",
			expect:     "[INFO]  test: this is a test\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer

			logger := New(&LoggerOptions{
				Name:   "test",
				Output: &stderr,
				Level:  Trace,
			})

			s := &stdlogAdapter{
				log:         logger,
				forceLevel:  c.forceLevel,
				inferLevels: c.inferLevels,
			}

			_, err := s.Write([]byte(c.write))
			assert.NoError(t, err)

			errStr := stderr.String()
			errDataIdx := strings.IndexByte(errStr, ' ')
			errRest := errStr[errDataIdx+1:]

			assert.Equal(t, c.expect, errRest)
		})
	}
}

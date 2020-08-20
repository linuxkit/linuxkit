package hclog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	t.Run("uses default output if none is given", func(t *testing.T) {
		var buf bytes.Buffer
		DefaultOutput = &buf

		logger := New(&LoggerOptions{
			Name: "test",
		})

		logger.Info("this is test", "who", "programmer", "why", "testing")

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=testing\n", rest)
	})

	t.Run("formats log entries", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("this is test", "who", "programmer", "why", "testing")

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=testing\n", rest)
	})

	t.Run("quotes values with spaces", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("this is test", "who", "programmer", "why", "testing is fun")

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=\"testing is fun\"\n", rest)
	})

	t.Run("renders slice values specially", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("this is test", "who", "programmer", "why", []interface{}{"testing", "dev", 1, uint64(5), []int{3, 4}})

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=[testing, dev, 1, 5, \"[3 4]\"]\n", rest)
	})

	t.Run("renders values in slices with quotes if needed", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("this is test", "who", "programmer", "why", []string{"testing & qa", "dev"})

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=[\"testing & qa\", dev]\n", rest)
	})

	t.Run("outputs stack traces", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("who", "programmer", "why", "testing", Stacktrace())

		lines := strings.Split(buf.String(), "\n")
		require.True(t, len(lines) > 1)

		assert.Equal(t, "github.com/hashicorp/go-hclog.Stacktrace", lines[1])
	})

	t.Run("outputs stack traces with it's given a name", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("who", "programmer", "why", "testing", "foo", Stacktrace())

		lines := strings.Split(buf.String(), "\n")
		require.True(t, len(lines) > 1)

		assert.Equal(t, "github.com/hashicorp/go-hclog.Stacktrace", lines[1])
	})

	t.Run("includes the caller location", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:            "test",
			Output:          &buf,
			IncludeLocation: true,
		})

		logger.Info("this is test", "who", "programmer", "why", "testing is fun")

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		// This test will break if you move this around, it's line dependent, just fyi
		assert.Equal(t, "[INFO]  go-hclog/logger_test.go:146: test: this is test: who=programmer why=\"testing is fun\"\n", rest)
	})

	t.Run("prefixes the name", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			// No name!
			Output: &buf,
		})

		logger.Info("this is test")
		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]
		assert.Equal(t, "[INFO]  this is test\n", rest)

		buf.Reset()

		another := logger.Named("sublogger")
		another.Info("this is test")
		str = buf.String()
		dataIdx = strings.IndexByte(str, ' ')
		rest = str[dataIdx+1:]
		assert.Equal(t, "[INFO]  sublogger: this is test\n", rest)
	})

	t.Run("use a different time format", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			TimeFormat: time.Kitchen,
		})

		logger.Info("this is test", "who", "programmer", "why", "testing is fun")

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')

		assert.Equal(t, str[:dataIdx], time.Now().Format(time.Kitchen))
	})

	t.Run("use with", func(t *testing.T) {
		var buf bytes.Buffer

		rootLogger := New(&LoggerOptions{
			Name:   "with_test",
			Output: &buf,
		})

		// Build the root logger in two steps, which triggers a slice capacity increase
		// and is part of the test for inadvertant slice aliasing.
		rootLogger = rootLogger.With("a", 1, "b", 2)
		rootLogger = rootLogger.With("c", 3)

		// Derive two new loggers which should be completely independent
		derived1 := rootLogger.With("cat", 30)
		derived2 := rootLogger.With("dog", 40)

		derived1.Info("test1")
		output := buf.String()
		dataIdx := strings.IndexByte(output, ' ')
		assert.Equal(t, "[INFO]  with_test: test1: a=1 b=2 c=3 cat=30\n", output[dataIdx+1:])

		buf.Reset()

		derived2.Info("test2")
		output = buf.String()
		dataIdx = strings.IndexByte(output, ' ')
		assert.Equal(t, "[INFO]  with_test: test2: a=1 b=2 c=3 dog=40\n", output[dataIdx+1:])
	})

	t.Run("unpaired with", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		var buf bytes.Buffer

		rootLogger := New(&LoggerOptions{
			Name:   "with_test",
			Output: &buf,
		})

		rootLogger = rootLogger.With("a")
	})

	t.Run("use with and log", func(t *testing.T) {
		var buf bytes.Buffer

		rootLogger := New(&LoggerOptions{
			Name:   "with_test",
			Output: &buf,
		})

		// Build the root logger in two steps, which triggers a slice capacity increase
		// and is part of the test for inadvertant slice aliasing.
		rootLogger = rootLogger.With("a", 1, "b", 2)
		// This line is here to test that when calling With with the same key,
		// only the last value remains (see issue #21)
		rootLogger = rootLogger.With("c", 4)
		rootLogger = rootLogger.With("c", 3)

		// Derive another logger which should be completely independent of rootLogger
		derived := rootLogger.With("cat", 30)

		rootLogger.Info("root_test", "bird", 10)
		output := buf.String()
		dataIdx := strings.IndexByte(output, ' ')
		assert.Equal(t, "[INFO]  with_test: root_test: a=1 b=2 c=3 bird=10\n", output[dataIdx+1:])

		buf.Reset()

		derived.Info("derived_test")
		output = buf.String()
		dataIdx = strings.IndexByte(output, ' ')
		assert.Equal(t, "[INFO]  with_test: derived_test: a=1 b=2 c=3 cat=30\n", output[dataIdx+1:])
	})

	t.Run("use with and log and change levels", func(t *testing.T) {
		var buf bytes.Buffer

		rootLogger := New(&LoggerOptions{
			Name:   "with_test",
			Output: &buf,
			Level:  Warn,
		})

		// Build the root logger in two steps, which triggers a slice capacity increase
		// and is part of the test for inadvertant slice aliasing.
		rootLogger = rootLogger.With("a", 1, "b", 2)
		rootLogger = rootLogger.With("c", 3)

		// Derive another logger which should be completely independent of rootLogger
		derived := rootLogger.With("cat", 30)

		rootLogger.Info("root_test", "bird", 10)
		output := buf.String()
		if output != "" {
			t.Fatalf("unexpected output: %s", output)
		}

		buf.Reset()

		derived.Info("derived_test")
		output = buf.String()
		if output != "" {
			t.Fatalf("unexpected output: %s", output)
		}

		derived.SetLevel(Info)

		rootLogger.Info("root_test", "bird", 10)
		output = buf.String()
		dataIdx := strings.IndexByte(output, ' ')
		assert.Equal(t, "[INFO]  with_test: root_test: a=1 b=2 c=3 bird=10\n", output[dataIdx+1:])

		buf.Reset()

		derived.Info("derived_test")
		output = buf.String()
		dataIdx = strings.IndexByte(output, ' ')
		assert.Equal(t, "[INFO]  with_test: derived_test: a=1 b=2 c=3 cat=30\n", output[dataIdx+1:])
	})

	t.Run("supports Printf style expansions when requested", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: &buf,
		})

		logger.Info("this is test", "production", Fmt("%d beans/day", 12))

		str := buf.String()
		dataIdx := strings.IndexByte(str, ' ')
		rest := str[dataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: production=\"12 beans/day\"\n", rest)
	})
}

func TestLogger_leveledWriter(t *testing.T) {
	t.Run("writes errors to stderr", func(t *testing.T) {
		var stderr bytes.Buffer
		var stdout bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: NewLeveledWriter(&stdout, map[Level]io.Writer{Error: &stderr}),
		})

		logger.Error("this is an error", "who", "programmer", "why", "testing")

		errStr := stderr.String()
		errDataIdx := strings.IndexByte(errStr, ' ')
		errRest := errStr[errDataIdx+1:]

		assert.Equal(t, "[ERROR] test: this is an error: who=programmer why=testing\n", errRest)
	})

	t.Run("writes non-errors to stdout", func(t *testing.T) {
		var stderr bytes.Buffer
		var stdout bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: NewLeveledWriter(&stdout, map[Level]io.Writer{Error: &stderr}),
		})

		logger.Info("this is test", "who", "programmer", "why", "testing")

		outStr := stdout.String()
		outDataIdx := strings.IndexByte(outStr, ' ')
		outRest := outStr[outDataIdx+1:]

		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=testing\n", outRest)
	})

	t.Run("writes errors and non-errors correctly", func(t *testing.T) {
		var stderr bytes.Buffer
		var stdout bytes.Buffer

		logger := New(&LoggerOptions{
			Name:   "test",
			Output: NewLeveledWriter(&stdout, map[Level]io.Writer{Error: &stderr}),
		})

		logger.Info("this is test", "who", "programmer", "why", "testing")
		logger.Error("this is an error", "who", "programmer", "why", "testing")

		errStr := stderr.String()
		errDataIdx := strings.IndexByte(errStr, ' ')
		errRest := errStr[errDataIdx+1:]

		outStr := stdout.String()
		outDataIdx := strings.IndexByte(outStr, ' ')
		outRest := outStr[outDataIdx+1:]

		assert.Equal(t, "[ERROR] test: this is an error: who=programmer why=testing\n", errRest)
		assert.Equal(t, "[INFO]  test: this is test: who=programmer why=testing\n", outRest)
	})
}

func TestLogger_JSON(t *testing.T) {
	t.Run("json formatting", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})

		logger.Info("this is test", "who", "programmer", "why", "testing is fun")

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, "programmer", raw["who"])
		assert.Equal(t, "testing is fun", raw["why"])
	})

	t.Run("json formatting with", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})
		logger = logger.With("cat", "in the hat", "dog", 42)

		logger.Info("this is test", "who", "programmer", "why", "testing is fun")

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, "programmer", raw["who"])
		assert.Equal(t, "testing is fun", raw["why"])
		assert.Equal(t, "in the hat", raw["cat"])
		assert.Equal(t, float64(42), raw["dog"])
	})

	t.Run("json formatting error type", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})

		errMsg := errors.New("this is an error")
		logger.Info("this is test", "who", "programmer", "err", errMsg)

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, "programmer", raw["who"])
		assert.Equal(t, errMsg.Error(), raw["err"])
	})

	t.Run("json formatting custom error type json marshaler", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})

		errMsg := &customErrJSON{"this is an error"}
		rawMsg, err := errMsg.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		expectedMsg, err := strconv.Unquote(string(rawMsg))
		if err != nil {
			t.Fatal(err)
		}

		logger.Info("this is test", "who", "programmer", "err", errMsg)

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, "programmer", raw["who"])
		assert.Equal(t, expectedMsg, raw["err"])
	})

	t.Run("json formatting custom error type text marshaler", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})

		errMsg := &customErrText{"this is an error"}
		rawMsg, err := errMsg.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		expectedMsg := string(rawMsg)

		logger.Info("this is test", "who", "programmer", "err", errMsg)

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, "programmer", raw["who"])
		assert.Equal(t, expectedMsg, raw["err"])
	})

	t.Run("supports Printf style expansions when requested", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})

		logger.Info("this is test", "production", Fmt("%d beans/day", 12))

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, "12 beans/day", raw["production"])
	})

	t.Run("includes the caller location", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:            "test",
			Output:          &buf,
			JSONFormat:      true,
			IncludeLocation: true,
		})

		logger.Info("this is test")
		_, file, line, ok := runtime.Caller(0)
		require.True(t, ok)

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, fmt.Sprintf("%v:%d", file, line-1), raw["@caller"])

	})

	t.Run("handles non-serializable entries", func(t *testing.T) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:       "test",
			Output:     &buf,
			JSONFormat: true,
		})

		myfunc := func() int { return 42 }
		logger.Info("this is test", "production", myfunc)

		b := buf.Bytes()

		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "this is test", raw["@message"])
		assert.Equal(t, errJsonUnsupportedTypeMsg, raw["@warn"])
	})
}

type customErrJSON struct {
	Message string
}

// error impl.
func (c *customErrJSON) Error() string {
	return c.Message
}

// json.Marshaler impl.
func (c customErrJSON) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(fmt.Sprintf("json-marshaler: %s", c.Message))), nil
}

type customErrText struct {
	Message string
}

// error impl.
func (c *customErrText) Error() string {
	return c.Message
}

// text.Marshaler impl.
func (c customErrText) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("text-marshaler: %s", c.Message)), nil
}

func BenchmarkLogger(b *testing.B) {
	b.Run("info with 10 pairs", func(b *testing.B) {
		var buf bytes.Buffer

		logger := New(&LoggerOptions{
			Name:            "test",
			Output:          &buf,
			IncludeLocation: true,
		})

		for i := 0; i < b.N; i++ {
			logger.Info("this is some message",
				"name", "foo",
				"what", "benchmarking yourself",
				"why", "to see what's slow",
				"k4", "value",
				"k5", "value",
				"k6", "value",
				"k7", "value",
				"k8", "value",
				"k9", "value",
				"k10", "value",
			)
		}
	})
}

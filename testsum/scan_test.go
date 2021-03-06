package testsum

import (
	"bytes"
	"strings"
	"testing"

	"time"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanNoFailures(t *testing.T) {
	source := `=== RUN   TestRunCommandSuccess
--- PASS: TestRunCommandSuccess (0.00s)
=== RUN   TestRunCommandWithCombined
--- PASS: TestRunCommandWithCombined (0.00s)
=== RUN   TestRunCommandWithTimeoutFinished
--- PASS: TestRunCommandWithTimeoutFinished (0.00s)
=== RUN   TestRunCommandWithTimeoutKilled
--- PASS: TestRunCommandWithTimeoutKilled (1.25s)
=== RUN   TestRunCommandWithErrors
--- PASS: TestRunCommandWithErrors (0.00s)
=== RUN   TestRunCommandWithStdoutStderr
--- PASS: TestRunCommandWithStdoutStderr (0.00s)
=== RUN   TestRunCommandWithStdoutStderrError
--- PASS: TestRunCommandWithStdoutStderrError (0.00s)
=== RUN   TestSkippedBecauseSomething
--- SKIP: TestSkippedBecauseSomething (0.00s)
        scan_test.go:39: becausde blah
PASS
ok      github.com/gotestyourself/gotestyourself/icmd   1.256s
`

	out := new(bytes.Buffer)
	summary, err := Scan(strings.NewReader(source), out)
	require.NoError(t, err)
	assert.NotZero(t, summary.Elapsed)
	summary.Elapsed = 0 // ignore elapsed
	assert.Equal(t, &Summary{Total: 8, Skipped: 1}, summary)
	assert.Equal(t, source, out.String())
}

func TestScanWithFailure(t *testing.T) {
	source := `=== RUN   TestRunCommandWithCombined
--- PASS: TestRunCommandWithCombined (0.00s)
=== RUN   TestRunCommandWithStdoutStderrError
--- PASS: TestRunCommandWithStdoutStderrError (0.00s)
=== RUN   TestThisShouldFail
Some output
More output
--- FAIL: TestThisShouldFail (0.00s)
        dummy_test.go:11: test is bad
        dummy_test.go:12: another failure
FAIL
exit status 1
FAIL    github.com/gotestyourself/gotestyourself/testsum        0.002s
`

	out := new(bytes.Buffer)
	summary, err := Scan(strings.NewReader(source), out)
	require.NoError(t, err)
	assert.NotZero(t, summary.Elapsed)
	summary.Elapsed = 0 // ignore elapsed
	assert.Equal(t, source, out.String())

	expected := &Summary{
		Total: 3,
		Failures: []Failure{
			{
				name:   "TestThisShouldFail",
				output: "Some output\nMore output\n",
				logs:   "        dummy_test.go:11: test is bad\n        dummy_test.go:12: another failure\n",
			},
		},
	}
	assert.Equal(t, expected, summary)
}

func TestScanWithNested(t *testing.T) {
	source := `=== RUN   TestNested
=== RUN   TestNested/a
=== RUN   TestNested/b
=== RUN   TestNested/c
--- PASS: TestNested (0.00s)
    --- PASS: TestNested/a (0.00s)
        dummy_test.go:27: Doing something for a
    --- PASS: TestNested/b (0.00s)
        dummy_test.go:27: Doing something for b
    --- PASS: TestNested/c (0.00s)
        dummy_test.go:27: Doing something for c
PASS
`

	summary, err := Scan(strings.NewReader(source), ioutil.Discard)
	require.NoError(t, err)
	assert.NotZero(t, summary.Elapsed)
	summary.Elapsed = 0 // ignore elapsed

	expected := &Summary{Total: 1}
	assert.Equal(t, expected, summary)
}

func TestScanWithNestedFailures(t *testing.T) {
	source := `=== RUN   TestNested
=== RUN   TestNested/a
Output from  a
=== RUN   TestNested/b
Output from  b
=== RUN   TestNested/c
Output from  c
--- FAIL: TestNested (0.00s)
    --- FAIL: TestNested/a (0.00s)
        dummy_test.go:28: Doing something for a
    --- FAIL: TestNested/b (0.00s)
        dummy_test.go:28: Doing something for b
    --- FAIL: TestNested/c (0.00s)
        dummy_test.go:28: Doing something for c
FAIL
exit status 1
`

	summary, err := Scan(strings.NewReader(source), ioutil.Discard)
	require.NoError(t, err)
	assert.NotZero(t, summary.Elapsed)
	summary.Elapsed = 0 // ignore elapsed

	expectedOutput := `=== RUN   TestNested/a
Output from  a
=== RUN   TestNested/b
Output from  b
=== RUN   TestNested/c
Output from  c
`
	expectedLogs := `    --- FAIL: TestNested/a (0.00s)
        dummy_test.go:28: Doing something for a
    --- FAIL: TestNested/b (0.00s)
        dummy_test.go:28: Doing something for b
    --- FAIL: TestNested/c (0.00s)
        dummy_test.go:28: Doing something for c
`

	expected := &Summary{
		Total: 1,
		Failures: []Failure{
			{name: "TestNested", output: expectedOutput, logs: expectedLogs},
		},
	}
	assert.Equal(t, expected, summary)
}

func TestSummaryFormatLine(t *testing.T) {
	var testcases = []struct {
		summary  Summary
		expected string
	}{
		{
			summary:  Summary{Total: 15, Elapsed: time.Minute},
			expected: "======== 15 tests in 60.00 seconds ========",
		},
		{
			summary:  Summary{Total: 100, Skipped: 3},
			expected: "======== 100 tests, 3 skipped in 0.00 seconds ========",
		},
		{
			summary: Summary{
				Total:    100,
				Failures: []Failure{{}},
				Elapsed:  3555 * time.Millisecond,
			},
			expected: "======== 100 tests, 1 failed in 3.56 seconds ========",
		},
		{
			summary: Summary{
				Total:    100,
				Skipped:  3,
				Failures: []Failure{{}},
				Elapsed:  42,
			},
			expected: "======== 100 tests, 3 skipped, 1 failed in 0.00 seconds ========",
		},
	}

	for _, testcase := range testcases {
		assert.Equal(t, testcase.expected, testcase.summary.FormatLine())
	}
}

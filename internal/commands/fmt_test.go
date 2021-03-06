package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/splunk/qbec/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsYaml(t *testing.T) {
	var tests = []struct {
		fileName string
		expected bool
	}{
		{"testdata/qbec.yaml", true},
		{"testdata/test.yml", true},
		{"testdata", false},
		{"testdata/components/c1.jsonnet", false},
		{"testdata/test.libsonnet", false},
	}
	for _, test := range tests {
		t.Run(test.fileName, func(t *testing.T) {
			f, err := os.Stat(test.fileName)
			if err != nil {
				t.Fatalf("Unexpected error'%v'", err)
			}
			var actual = isYamlFile(f)
			if test.expected != actual {
				t.Errorf("Expected '%t', got '%t'", test.expected, actual)
			}
		})
	}
}

func TestShouldFormat(t *testing.T) {
	var tests = []struct {
		fileName string
		config   fmtCommandConfig
		expected bool
	}{
		{"testdata/qbec.yaml", fmtCommandConfig{formatYaml: true}, true},
		{"testdata/test.yml", fmtCommandConfig{formatJsonnet: true}, false},
		{"testdata", fmtCommandConfig{formatYaml: true, formatJsonnet: true}, false},
		{"testdata/components/c1.jsonnet", fmtCommandConfig{formatJsonnet: true}, true},
	}
	for _, test := range tests {
		t.Run(test.fileName, func(t *testing.T) {
			f, err := os.Stat(test.fileName)
			if err != nil {
				t.Fatalf("Unexpected error'%v'", err)
			}
			var actual = shouldFormat(&test.config, f)
			if test.expected != actual {
				t.Errorf("Expected '%t', got '%t'", test.expected, actual)
			}

		})
	}
}
func TestIsJsonnet(t *testing.T) {
	var tests = []struct {
		fileName string
		expected bool
	}{
		{"testdata/components/c1.jsonnet", true},
		{"testdata/test.libsonnet", true},
		{"testdata", false},
		{"testdata/qbec.yaml", false},
		{"testdata/test.yml", false},
	}
	for _, test := range tests {
		t.Run(test.fileName, func(t *testing.T) {
			f, err := os.Stat(test.fileName)
			if err != nil {
				t.Fatalf("Unexpected error'%v'", err)
			}
			var actual = isJsonnetFile(f)
			if test.expected != actual {
				t.Errorf("Expected '%t', got '%t'", test.expected, actual)
			}
		})
	}
}

func TestDoFmt(t *testing.T) {
	var b bytes.Buffer
	var tests = []struct {
		args        []string
		config      fmtCommandConfig
		expectedErr string
	}{
		{[]string{}, fmtCommandConfig{check: true, write: true}, `check and write are not supported together`},
		{[]string{"nonexistentfile"}, fmtCommandConfig{}, testutil.FileNotFoundMessage},
		{[]string{"testdata/qbec.yaml"}, fmtCommandConfig{formatYaml: true, config: &config{stdout: &b}}, ""},
		{[]string{"testdata/components"}, fmtCommandConfig{formatJsonnet: true, config: &config{stdout: &b}}, ""},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			var err = doFmt(test.args, &test.config)
			if test.expectedErr == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
			}
		})
	}

}

func TestFormatYaml(t *testing.T) {
	var testfile, err = ioutil.ReadFile("testdata/test.yml")
	require.Nil(t, err)
	o, err := formatYaml(testfile)
	require.Nil(t, err)
	e, err := ioutil.ReadFile("testdata/test.yml.formatted")
	require.Nil(t, err)
	if !bytes.Equal(o, e) {
		t.Errorf("Expected %q, got %q", string(e), string(o))
	}

	var tests = []struct {
		input     []byte
		expectErr bool
	}{
		{input: nil, expectErr: false},
		{input: []byte("abc"), expectErr: false},
		{input: []byte("---"), expectErr: false},
		{input: []byte("---\nnull\n---"), expectErr: false},
		{input: []byte("*abc*"), expectErr: true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			var _, err = formatYaml(test.input)
			if test.expectErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestFormatJsonnet(t *testing.T) {
	var testfile, err = ioutil.ReadFile("testdata/test.libsonnet")
	require.Nil(t, err)
	o, err := formatJsonnet(testfile)
	require.Nil(t, err)
	e, err := ioutil.ReadFile("testdata/test.libsonnet.formatted")
	require.Nil(t, err)
	if !bytes.Equal(o, e) {
		t.Errorf("Expected %q, got %q", string(e), string(o))
	}
}

func TestFmtCommand(t *testing.T) {
	s := newScaffold(t)
	defer s.reset()
	err := s.executeCommand("alpha", "fmt", "--yaml", "prod-env.yaml")
	require.Nil(t, err)
	s.assertOutputLineMatch(regexp.MustCompile(`      - service2`))
}

func TestProcessFile(t *testing.T) {

	var tests = []struct {
		input  string
		output string
	}{
		{input: "testdata/test.libsonnet", output: "testdata/test.libsonnet.formatted"},
		{input: "testdata/test.yml", output: "testdata/test.yml.formatted"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var b bytes.Buffer
			var config = fmtCommandConfig{}
			var err = processFile(&config, test.input, nil, &b)
			require.Nil(t, err)
			e, err := ioutil.ReadFile(test.output)
			require.Nil(t, err)
			var o = b.Bytes()
			if !bytes.Equal(e, o) {
				t.Errorf("Expected %q, got %q", string(e), string(o))
			}
		})
	}
}

// Adapted from https://golang.org/src/cmd/gofmt/gofmt_test.go
func TestBackupFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "qbecfmt_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	name, err := backupFile(filepath.Join(dir, "foo.yaml"), []byte("a: 1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created: %s", name)
}

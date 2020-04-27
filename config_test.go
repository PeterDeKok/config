package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

type ECSomething struct {
	Blaat string
}

type ECAaaaa struct {
	Aa string
	Bb int
}

type ExpectedCnf struct {
	Something ECSomething
	Aaaaa     ECAaaaa
}

var (
	expectedCnf = &ExpectedCnf{
		Something: ECSomething{Blaat: "asdf"},
		Aaaaa:     ECAaaaa{Aa: "bbbbbbbb", Bb: 32},
	}
	expected2Cnf = &ExpectedCnf{
		Something: ECSomething{Blaat: "asdf2"},
		Aaaaa:     ECAaaaa{Aa: "bbbbbbbb2", Bb: 322},
	}
)

func TestInit(t *testing.T) {
	if cache == nil {
		t.Fatal("expected init to have created singleton")
	}

	if cache != Singleton() {
		t.Fatal("expected singleton method to NOT create new instance")
	}

	if cache == newCache() {
		t.Fatal("expected newCache method to NOT be new instance")
	}

	if len(cache.path) > 0 {
		t.Fatal("expected path to be empty")
	}

	if cache.cache == nil {
		t.Fatal("expected cache to be initialized with an empty slice: got nil")
	}

	if len(cache.cache) > 0 {
		t.Fatal("expected cache to be initialized with an empty slice: got a filled slice")
	}

	if cache.readers == nil {
		t.Fatal("expected readers to be initialized with an empty slice: got nil")
	}

	if len(cache.readers) > 0 {
		t.Fatal("expected readers to be initialized with an empty slice: got a filled slice")
	}
}

func TestLoadConfigWithoutPath(t *testing.T) {
	cache = newCache()

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("loadConfig without path should panic")
		}

		err, ok := r.(error)

		if !ok {
			t.Fatal("loadConfig panic value should be error")
		}

		var expected = "configuration not loaded, set '-c /my/path/config.toml'"

		if err.Error() != expected {
			t.Fatalf("expected loadConfig panic value to be: %s, got: %s", expected, err)
		}
	}()

	cache.loadConfig()
}

func TestLoadConfigWithInvalidPath(t *testing.T) {
	cache = newCache()

	cache.path = "testdata/non-existing.config.test.toml"

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("loadConfig with invalid path should panic")
		}

		err, ok := r.(error)

		if !ok {
			t.Fatal("loadConfig panic value should be error")
		}

		var expected = "configuration not loaded: open testdata/non-existing.config.test.toml: no such file or directory"

		if err.Error() != expected {
			t.Fatalf("expected loadConfig panic value to be: %s, got: %s", expected, err)
		}
	}()

	cache.loadConfig()
}

func TestConfigFileArg(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd"}

	if path := configFileArg(); path != "" {
		t.Fatalf("Expected empty path, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-s", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "" {
		t.Fatalf("Expected empty path, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-c=some/path", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "some/path" {
		t.Fatalf("Expected path to be: some/path, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-c=\"some/path\"", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "some/path" {
		t.Fatalf("Expected path to be: some/path, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-c='some/path'", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "some/path" {
		t.Fatalf("Expected path to be: some/path, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-c", "some/path2", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "some/path2" {
		t.Fatalf("Expected path to be: some/path2, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-c=p", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "p" {
		t.Fatalf("Expected path to be: p, got: %s", path)
	}

	os.Args = []string{"cmd", "somevalue", "-c", "p", "--someflag", "\"some encapsulated string\""}

	if path := configFileArg(); path != "p" {
		t.Fatalf("Expected path to be: p, got: %s", path)
	}
}

func TestLoadConfigWithPath(t *testing.T) {
	cache = newCache()
	cache.path = "testdata/config.test.toml"

	cache.loadConfig()

	if cache.cache == nil {
		t.Fatal("expected cache.cache to be initialized with a slice, got nil")
	}

	var expected = "[something]\n    blaat=\"asdf\"\n    qwer=\"qwerqwer\"\n\n[aaaaa]\n    aa=\"bbbbbbbb\"\n    bb=32"

	if string(cache.cache) != expected {
		t.Fatalf("expected cache.cache to be %s, got: %s", expected, string(cache.cache))
	}
}

func TestEnsureConfigLoadedWithoutPathOrArg(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd"}

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("ensureConfigLoaded with invalid path should panic")
		}

		err, ok := r.(error)

		if !ok {
			t.Fatal("ensureConfigLoaded panic value should be error")
		}

		var expected = "configuration not loaded, set '-c /my/path/config.toml'"

		if err.Error() != expected {
			t.Fatalf("expected ensureConfigLoaded panic value to be: %s, got: %s", expected, err)
		}
	}()

	cache.ensureConfigLoaded(false)
}

func TestEnsureConfigLoadedWithoutPathWithNonExistingArg(t *testing.T) {
	oldArgs := os.Args
	oldFilepathAbs := filepathAbs
	defer func() {
		os.Args = oldArgs
		filepathAbs = oldFilepathAbs
	}()

	os.Args = []string{"cmd", "-c", "testdata/non-existing.config.test.toml"}
	filepathAbs = func(path string) (string, error) {
		return "", errors.New("mocked error")
	}

	cache = newCache()

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("ensureConfigLoaded with invalid path should panic")
		}

		err, ok := r.(error)

		if !ok {
			t.Fatal("ensureConfigLoaded panic value should be error")
		}

		var expected = "could not decode config path: mocked error"

		if err.Error() != expected {
			t.Fatalf("expected ensureConfigLoaded panic value to be: %s, got: %s", expected, err)
		}
	}()

	cache.ensureConfigLoaded(false)
}

func TestEnsureConfigLoadedWithoutPathWithArg(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd", "-c", "testdata/config2.test.toml"}

	cache.ensureConfigLoaded(false)

	if cache.cache == nil {
		t.Fatal("expected cache.cache to be initialized with a slice, got nil")
	}

	var expected = "[something]\n    blaat=\"asdf2\"\n    qwer=\"qwerqwer2\"\n\n[aaaaa]\n    aa=\"bbbbbbbb2\"\n    bb=322"

	if string(cache.cache) != expected {
		t.Fatalf("expected cache.cache to be %s, got: %s", expected, string(cache.cache))
	}

	var err error

	cache.path, err = filepath.Abs("testdata/config.test.toml")

	if err != nil {
		panic(err)
	}

	cache.ensureConfigLoaded(true)

	if cache.cache == nil {
		t.Fatal("expected cache.cache to be initialized with a slice, got nil")
	}

	expected = "[something]\n    blaat=\"asdf\"\n    qwer=\"qwerqwer\"\n\n[aaaaa]\n    aa=\"bbbbbbbb\"\n    bb=32"

	if string(cache.cache) != expected {
		t.Fatalf("expected cache.cache to be %s, got: %s", expected, string(cache.cache))
	}
}

func TestParseConfigNoCache(t *testing.T) {
	cache = newCache()

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("ensureConfigLoaded with invalid path should panic")
		}

		err, ok := r.(error)

		if !ok {
			t.Fatal("ensureConfigLoaded panic value should be error")
		}

		var expected = "configuration not loaded, set '-c /my/path/config.toml'"

		if err.Error() != expected {
			t.Fatalf("expected ensureConfigLoaded panic value to be: %s, got: %s", expected, err)
		}
	}()

	cache.parseConfig(nil)
}

func TestParseConfigInvalidToml(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd", "-c", "testdata/config-invalid.test.toml"}

	cache.ensureConfigLoaded(false)

	if cache.cache == nil {
		t.Fatal("expected cache.cache to be initialized with a slice, got nil")
	}

	var expected = "[something\n    blaat=\"asdf2\"\n    qwer=\"qwerqwer2\"\n\n[aaaaa]\n    aa=\"bbbbbbbb2\"\n    bb=322"

	if string(cache.cache) != expected {
		t.Fatalf("expected cache.cache to be %s, got: %s", expected, string(cache.cache))
	}

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("ensureConfigLoaded with invalid path should panic")
		}

		err, ok := r.(error)

		if !ok {
			t.Fatal("ensureConfigLoaded panic value should be error")
		}

		var expected = "line 2: invalid TOML syntax"

		if err.Error() != expected {
			t.Fatalf("expected ensureConfigLoaded panic value to be: %s, got: %s", expected, err)
		}
	}()

	cache.parseConfig(nil)
}

func TestParseConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd", "-c", "testdata/config.test.toml"}

	cache.ensureConfigLoaded(false)

	if cache.cache == nil {
		t.Fatal("expected cache.cache to be initialized with a slice, got nil")
	}

	var expected = "[something]\n    blaat=\"asdf\"\n    qwer=\"qwerqwer\"\n\n[aaaaa]\n    aa=\"bbbbbbbb\"\n    bb=32"

	if string(cache.cache) != expected {
		t.Fatalf("expected cache.cache to be %s, got: %s", expected, string(cache.cache))
	}

	actual := &ExpectedCnf{}

	cache.parseConfig(actual)

	if !reflect.DeepEqual(actual, expectedCnf) {
		t.Fatalf("expected cnf to be %v, got: %v", expectedCnf, actual)
	}
}

func TestAddReader(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd", "-c", "testdata/config.test.toml"}

	actual := &ExpectedCnf{}

	cache.Add(actual)

	if !reflect.DeepEqual(actual, expectedCnf) {
		t.Fatalf("expected cnf to be %v, got: %v", expectedCnf, actual)
	}
}

func TestReload(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd", "-c", "testdata/config.test.toml"}

	actual := &ExpectedCnf{}

	cache.Add(actual)

	var err error

	cache.path, err = filepath.Abs("testdata/config2.test.toml")

	if err != nil {
		panic(err)
	}

	cache.Reload()

	if !reflect.DeepEqual(actual, expected2Cnf) {
		t.Fatalf("expected cnf to be %v, got: %v", expected2Cnf, actual)
	}
}

func TestReloadBySignal(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cache = newCache()
	os.Args = []string{"cmd", "-c", "testdata/config.test.toml"}

	actual := &ExpectedCnf{}

	cache.Add(actual)

	var err error

	cache.path, err = filepath.Abs("testdata/config2.test.toml")

	if err != nil {
		panic(err)
	}

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR1); err != nil {
		panic(err)
	}

	// Kill is an intrinsic async action, wait a while to ensure it has triggered
	time.Sleep(10 * time.Millisecond)

	if !reflect.DeepEqual(actual, expected2Cnf) {
		t.Fatalf("expected cnf to be %v, got: %v", expected2Cnf, actual)
	}
}

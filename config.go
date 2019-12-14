// Package config provides tools to easily add (TOML) config files to packages
//
// The config package will load the toml file on initialization and cache it.
// This cache will be used to populate any config structs added to the pool by different packages.
// On reload (signal SIGUSR1) the file is reloaded, the cache refilled
// and every entry in the pool parsed again.
//
// The file should be specified by adding -c to the command
package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/naoina/toml"
	"io/ioutil"
	"log"
	"path/filepath"
	"peterdekok.nl/gotools/trap"
	"reflect"
	"sync"
)

var (
	tomlCnf *toml.Config

	path    string
	cnfToml []byte

	cnfs []interface{}
	mux  sync.RWMutex
)

func init() {
	mux.Lock()
	defer mux.Unlock()

	var (
		ipath string
		err   error
	)

	// TODO This form of retrieving the config file location is a bit ugly...
	flag.StringVar(&ipath, "c", "", "Path to the configuration file, see: config.example.toml")
	flag.Parse()

	if ipath == "" {
		err := errors.New("configuration path not specified, set -c=/my/path/config.toml")

		// Just in case the logger does not panic/exit
		panic(err)
	}

	// Get the absolute path, even when a relative path is given
	path, err = filepath.Abs(ipath)
	if err != nil {
		log.Fatalf("could not decode config path %v", err)
	}

	tomlCnf = &toml.Config{
		NormFieldName: toml.DefaultConfig.NormFieldName,
		FieldToKey:    toml.DefaultConfig.FieldToKey,
		MissingField:  func(typ reflect.Type, key string) error { return nil },
	}

	trap.OnReload(func() {
		Reload()
	})

	loadConfig()
}

// loadConfig will read the file
func loadConfig() {
	var err error

	// Load the toml to memory
	cnfToml, err = ioutil.ReadFile(path)

	if err != nil {
		panic(err)
	}
}

// parseConfig decodes the cached file content and parse it into the individual config struct
func parseConfig(sss interface{}) {
	if err := tomlCnf.NewDecoder(bytes.NewReader(cnfToml)).Decode(sss); err != nil {
		panic(err)
	}
}

// Reload the config file and parse for every individual config struct
func Reload() {
	mux.Lock()
	defer mux.Unlock()

	fmt.Println("About to reload config")

	loadConfig()

	for i, cnf := range cnfs {
		fmt.Printf("Reloading %d\n > current values: %s\n", i, cnf)

		parseConfig(cnf)

		fmt.Printf(" > new values: %s\n", cnf)
	}

	fmt.Println("Done reloading config")
}

// Add a config struct to the pool
func Add(cnf interface{}) {
	mux.Lock()
	defer mux.Unlock()

	cnfs = append(cnfs, cnf)

	parseConfig(cnf)
}

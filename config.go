// Package config provides tools to easily add (TOML) config files to packages
//
// The config package will load the toml file on initialization and cache it.
// This cache will be used to populate any config structs added to the pool by different packages.
// On reload (signal SIGUSR1) the file is reloaded, the cache refilled
// and every entry in the pool parsed again.
//
// The file should be specified by adding a -c flag/argument to the command
package config

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/naoina/toml"
	"io/ioutil"
	"os"
	"path/filepath"
	"peterdekok.nl/gotools/trap"
	"reflect"
	"strings"
	"sync"
)

type Cache struct {
	sync.RWMutex
	cache []byte

	path            string
	decoderSettings *toml.Config

	readers []interface{}
}

var (
	cache *Cache
	once  sync.Once

	filepathAbs = filepath.Abs
)

func init() {
	cache = Singleton()
}

func Singleton() *Cache {
	once.Do(func() {
		cache = newCache()

		trap.OnReload(func() { cache.Reload() })
	})

	return cache
}

// Internal constructor for (singleton) cache instance.
// Only separated from Singleton method for testing.
func newCache() *Cache {
	return &Cache{
		cache: make([]byte, 0),

		path: "",

		decoderSettings: &toml.Config{
			NormFieldName: toml.DefaultConfig.NormFieldName,
			FieldToKey:    toml.DefaultConfig.FieldToKey,
			MissingField:  func(typ reflect.Type, key string) error { return nil },
		},

		readers: make([]interface{}, 0),
	}
}

// Reload the config file and parse for every individual config struct
func (c *Cache) Reload() {
	c.Lock()
	defer c.Unlock()

	fmt.Println("About to reload config")

	c.ensureConfigLoaded(true)

	for i, reader := range c.readers {
		fmt.Printf("Reloading %d\n > current values: %s\n", i, reader)

		c.parseConfig(reader)

		fmt.Printf(" > new values: %s\n", reader)
	}

	fmt.Println("Done reloading config")
}

// Add a config struct to the pool
func (c *Cache) Add(cnf interface{}) {
	c.Lock()
	defer c.Unlock()

	c.readers = append(c.readers, cnf)

	c.ensureConfigLoaded(false)

	c.parseConfig(cnf)
}

// EnsureConfigLoaded
// - reads the file location from the command arguments if not read
// - loads config from file if not loaded
func (c *Cache) ensureConfigLoaded(reload bool) {
	if len(c.path) == 0 {
		if ipath := configFileArg(); len(ipath) > 0 {
			var err error

			// Get the absolute path, even when a relative path is given
			c.path, err = filepathAbs(ipath)

			if err != nil {
				panic(fmt.Errorf("could not decode config path: %v", err))
			}
		}
	}

	if len(c.path) == 0 {
		panic(fmt.Errorf("configuration not loaded, set '-c /my/path/config.toml'"))
	}

	if len(c.cache) == 0 || reload {
		c.loadConfig()
	}
}

// loadConfig will read the config file and cache the bytes
func (c *Cache) loadConfig() {
	if len(c.path) == 0 {
		panic(errors.New("configuration not loaded, set '-c /my/path/config.toml'"))
	}

	var err error

	// Load the toml to memory
	c.cache, err = ioutil.ReadFile(c.path)

	if err != nil {
		panic(fmt.Errorf("configuration not loaded: %s", err))
	}
}

// parseConfig decodes the cached file content and parse it into the individual config struct
func (c *Cache) parseConfig(sss interface{}) {
	if len(c.cache) == 0 {
		panic(errors.New("configuration not loaded, set '-c /my/path/config.toml'"))
	}

	if err := c.decoderSettings.NewDecoder(bytes.NewReader(c.cache)).Decode(sss); err != nil {
		panic(err)
	}
}

// configFileArg tries to retrieve the config file location from the commandline arguments
func configFileArg() string {
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-c=") {
			arg = strings.TrimPrefix(arg, "-c=")
		} else if len(arg) == 0 || os.Args[i] != "-c" {
			continue
		}

		// From this point on, the current argument is (supposed to be) a config path
		if len(arg) == 1 {
			return arg
		}

		if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") {
			arg = arg[1 : len(arg)-1]
		}

		if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") {
			arg = arg[1 : len(arg)-1]
		}

		return arg
	}

	return ""
}

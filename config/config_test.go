package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type MyConfig struct {
	File    string `               short:"f" long:"config"  description:"Yaml config"`
	Verbose []bool `yaml:"verbose" short:"v" long:"verbose" description:"Show verbose debug information"`
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MyConfig) DeepCopyInto(out *MyConfig) {
	*out = *in
	if in.Verbose != nil {
		in, out := &in.Verbose, &out.Verbose
		*out = make([]bool, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MyConfig.
func (in *MyConfig) DeepCopy() *MyConfig {
	if in == nil {
		return nil
	}
	out := new(MyConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyConfig is an autogenerated deepcopy function, copying the receiver, creating a new Config.
func (in *MyConfig) DeepCopyConfig() Config {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// ConfigFile returns the path to the configuration file to parse.
func (in MyConfig) ConfigFile() string {
	return in.File
}

type testLogger struct {
	*testing.T
}

func (t *testLogger) Tracef(format string, vals ...interface{}) {
	t.Helper()
	t.Logf("go-libqd/config: "+format, vals...)
}

func (t *testLogger) Debugf(format string, vals ...interface{}) {
	t.Helper()
	t.Logf("go-libqd/config: "+format, vals...)
}

func (t *testLogger) Infof(format string, vals ...interface{}) {
	t.Helper()
	t.Logf("go-libqd/config: "+format, vals...)
}

func (t *testLogger) Warnf(format string, vals ...interface{}) {
	t.Helper()
	t.Logf("go-libqd/config: "+format, vals...)
}

func TestMyConfig(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create temporary file for config
	tmpFile, err := ioutil.TempFile(os.TempDir(), "libqd-config-")

	if err != nil {
		t.Error(err)
		return
	}

	defer os.Remove(tmpFile.Name())

	// Logger and test log wrapper
	logger := &testLogger{t}

	myConfig := &MyConfig{
		// We need to define it otherwise yaml.Marshal will set it to empty
		File:    tmpFile.Name(),
		Verbose: []bool{true, true, true},
	}

	yml, err := yaml.Marshal(myConfig)

	if err != nil {
		t.Error(err)
		return
	}

	err = ioutil.WriteFile(tmpFile.Name(), yml, 0)

	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(10 * time.Microsecond)

	// Override go test os.Args
	os.Args = []string{
		"test", "-vvvvvv", "-f", tmpFile.Name(),
	}

	// Some variable
	a := 0

	// Validator
	validator := func(currentConfig Config, newConfig Config) []error {
		var errs []error
		var ok bool
		var currentConf *MyConfig

		// currentConfig is nil the first time the validator is called
		if currentConfig != nil {
			// Casting currentConfig from Config to (*MyConfig)
			currentConf, ok = currentConfig.(*MyConfig)

			if !ok {
				errs = append(errs, fmt.Errorf("Can not cast currentConfig to MyConfig"))
				return errs
			}
		}

		// Casting newConfig from Config to (*MyConfig)
		newConf, ok := newConfig.(*MyConfig)

		if !ok {
			errs = append(errs, fmt.Errorf("Can not cast newConfig to MyConfig"))
			return errs
		}

		// ---------------------------------------------------------------------
		// Here begins the actual validation of the values of newConfig
		// ---------------------------------------------------------------------

		if len(newConf.Verbose) > 6 {
			errs = append(errs, fmt.Errorf("Verbose `%d` can not be greater than 6", len(newConf.Verbose)))
		}

		if currentConf != nil {
			if currentConf.File != newConf.File {
				errs = append(errs, fmt.Errorf("File `%s` can not be changed to `%s`", currentConf.File, newConf.File))
			}
		}

		return errs
	}

	// Applier
	applier := func(currentConfig Config, newConfig Config) error {
		var err error
		var ok bool
		var currentConf *MyConfig

		if currentConfig != nil {
			currentConf, ok = currentConfig.(*MyConfig)

			if !ok {
				return fmt.Errorf("Can not cast currentConfig to (*MyConfig)")
			}
		}

		newConf, ok := newConfig.(*MyConfig)

		if !ok {
			return fmt.Errorf("Can not cast newConfig to (*MyConfig)")
		}

		// Increment `a` only after first reload
		if currentConf != nil && newConf != nil {
			a++
		}

		return err
	}

	confManager := GetManager(logger)
	confManager.AddValidators(nil, validator)
	confManager.AddAppliers(nil, applier)

	// Launch config
	err = confManager.MakeConfig(ctx, nil, myConfig)

	if err != nil {
		t.Error(err)
		return
	}

	if a != 0 {
		t.Errorf("a=%d but should be 0", a)
	}

	m := confManager.GetConfig(nil).(*MyConfig)

	t.Logf("%#v", m)

	c := confManager.NewConfigChan(nil)

	yml, err = yaml.Marshal(myConfig)

	if err != nil {
		t.Error(err)
		return
	}

	err = ioutil.WriteFile(tmpFile.Name(), yml, 0)

	if err != nil {
		t.Error(err)
		return
	}

	// Check that a new config is sent via the channel
	select {
	case newConf := <-c:
		t.Logf("%#v", newConf)
	case <-time.After(5 * time.Second):
		t.Error("No new configuration received")
		return
	}

	if a != 1 {
		t.Errorf("a=%d but should be 1", a)
	}
}

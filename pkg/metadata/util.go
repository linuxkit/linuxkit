package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
	"github.com/sl1pm4t/snooze"
	"github.com/thoas/go-funk"
)

// ProviderUtil utility for providers
type ProviderUtil interface {
	ProviderShortName
	PrepareLogger() *logrus.Entry
}

// DefaultProviderUtil default method for provider utility, same as default interface/Type method in c#/rust
type DefaultProviderUtil struct {
	ProviderUtil
}

// PrepareLogger Prepare the logger to add contextual name for current provider
func (p *DefaultProviderUtil) PrepareLogger() *logrus.Entry {
	return logrus.WithField("provider", p.ShortName())
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

type simpleExtractData struct {
	Type    string
	Dest    string
	Perm    os.FileMode
	Getter  func() (string, error)
	Success func() error
}

func (p *DefaultProviderUtil) simpleExtract(bar []simpleExtractData) (err error) {
	var ret string
	for _, data := range bar {
		log := p.PrepareLogger().WithField("type", data.Type)
		if ret, err = data.Getter(); err == nil {
			if err = os.MkdirAll(filepath.Dir(data.Dest), data.Perm); err != nil {
				return
			}
			if err = ioutil.WriteFile(data.Dest, []byte(ret), data.Perm); err != nil {
				return
			}
			log.WithField("data", ret).Debug("wrote")
			if data.Success != nil {
				err = data.Success()
			}
		} else {
			log.WithError(err).
				Error("unable to get")
		}
	}
	return
}

func ensureSSHKeySecure() error {
	return os.Chmod(path.Join(ConfigPath, SSH, "authorized_keys"), 0600)
}

func snoozeSetDefaultLogger(client *snooze.Client) {
	oldBefore := client.Before
	client.Before = func(req *retryablehttp.Request, client_ *retryablehttp.Client) {
		client_.Logger = &logrusLeveledLogger{logrus.StandardLogger()}
		if oldBefore != nil {
			oldBefore(req, client_)
		}
	}
}

type logrusLeveledLogger struct {
	*logrus.Logger
}

func kvToLogrusFields(keysAndValues []interface{}) logrus.Fields {
	// it is a kv pair but in flatten array form so we re-pair them for every 2 elements
	return funk.Map(funk.Chunk(keysAndValues, 2), func(tpl []interface{}) (k string, v interface{}) {
		// we assume the keys are string anyway
		k, v = tpl[0].(string), tpl[1]
		return
	}).(map[string]interface{})
}

func (l *logrusLeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	l.WithFields(kvToLogrusFields(keysAndValues)).Error(msg)
}

func (l *logrusLeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	l.WithFields(kvToLogrusFields(keysAndValues)).Info(msg)
}

func (l *logrusLeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.WithFields(kvToLogrusFields(keysAndValues)).Debug(msg)
}

func (l *logrusLeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.WithFields(kvToLogrusFields(keysAndValues)).Warn(msg)
}

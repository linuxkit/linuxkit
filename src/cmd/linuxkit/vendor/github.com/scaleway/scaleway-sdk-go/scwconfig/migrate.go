package scwconfig

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/scaleway/scaleway-sdk-go/logger"
	"gopkg.in/yaml.v2"
)

const (
	defaultConfigPermission = 0600
)

// migrateV1toV2 converts the V1 config to V2 config and save it in the target path
// use config.Save() when the method is public
func migrateV1toV2(configV1 *configV1, targetPath string) error {
	// STEP 0: get absolute target path

	targetPath = filepath.Clean(targetPath)

	// STEP 1: create dir
	err := os.MkdirAll(filepath.Dir(targetPath), 0700)
	if err != nil {
		logger.Debugf("mkdir did not work on %s: %s", filepath.Dir(targetPath), err)
		return nil
	}

	// STEP 2: marshal yaml config
	newConfig := configV1.toV2()
	file, err := yaml.Marshal(newConfig)
	if err != nil {
		return err
	}

	// STEP 3: save config
	err = ioutil.WriteFile(targetPath, file, defaultConfigPermission)
	if err != nil {
		logger.Debugf("cannot write file %s: %s", targetPath, err)
		return nil
	}

	// STEP 4: log success
	logger.Infof("config successfully migrated to %s", targetPath)
	return nil
}

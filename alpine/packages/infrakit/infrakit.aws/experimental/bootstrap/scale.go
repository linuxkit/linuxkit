package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
)

func scale(cluster clusterID, groupName string, count int) (*fakeSWIMSchema, error) {
	log.Infoln("Fetching from", cluster.url())
	resp, err := http.Get(cluster.url())
	if err != nil {
		abort("Failed to fetch current configuration: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response when fetching configuration: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to fetch current configuration: %s", string(body))
	}

	swim := fakeSWIMSchema{}
	err = json.Unmarshal(body, &swim)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse existing configuration: %s", err)
	}

	matched := false
	swim.mutateGroups(func(group *instanceGroup) {
		if string(group.Name) == groupName {
			matched = true
			if group.isManager() {
				err = errors.New("A manager group may not be scaled")
			}

			group.Size = count
		}
	})
	if err != nil {
		return nil, err
	}

	if !matched {
		log.Fatalf("Config does not contain a group named %s", groupName)
		os.Exit(1)
	}

	err = swim.push()
	if err != nil {
		return nil, fmt.Errorf("Failed to push config: %s", err)
	}

	log.Infof("Target count for group %s is now %d", groupName, count)
	return &swim, nil
}

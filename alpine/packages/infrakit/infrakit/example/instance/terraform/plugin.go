package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/spi/instance"
	"github.com/nightlyone/lockfile"
	"github.com/spf13/afero"
)

// This example uses terraform as the instance plugin.
// It is very similar to the file instance plugin.  When we
// provision an instance, we write a *.tf.json file in the directory
// and call terra apply.  For describing instances, we parse the
// result of terra show.  Destroying an instance is simply removing a
// tf.json file and call terra apply again.

type plugin struct {
	Dir       string
	fs        afero.Fs
	lock      lockfile.Lockfile
	applying  bool
	applyLock sync.Mutex
}

// NewTerraformInstancePlugin returns an instance plugin backed by disk files.
func NewTerraformInstancePlugin(dir string) instance.Plugin {
	log.Debugln("terraform instance plugin. dir=", dir)
	lock, err := lockfile.New(filepath.Join(dir, "tf-apply.lck"))
	if err != nil {
		panic(err)
	}

	return &plugin{
		Dir:  dir,
		fs:   afero.NewOsFs(),
		lock: lock,
	}
}

/*
TFormat models the on disk representation of a terraform resource JSON.

An example of this looks like:

{
    "resource" : {
	"aws_instance" : {
	    "web4" : {
		"ami" : "${lookup(var.aws_amis, var.aws_region)}",
		"instance_type" : "m1.small",
		"key_name": "PUBKEY",
		"vpc_security_group_ids" : ["${aws_security_group.default.id}"],
		"subnet_id": "${aws_subnet.default.id}",
		"tags" :  {
		    "Name" : "web4",
		    "InstancePlugin" : "terraform"
		}
		"connection" : {
		    "user" : "ubuntu"
		},
		"provisioner" : {
		    "remote_exec" : {
			"inline" : [
			    "sudo apt-get -y update",
			    "sudo apt-get -y install nginx",
			    "sudo service nginx start"
			]
		    }
		}
	    }
	}
    }
}

Note that the JSON above has a name (web4).  In general, we do not require names to
be specified. So this means the raw JSON we support needs to omit the name. So the instance.Spec
JSON looks like below, where the value of `value` is the instance body of the TF format JSON.

{
    "Properties" : {
        "type" : "aws_instance",
        "value" : {
            "ami" : "${lookup(var.aws_amis, var.aws_region)}",
            "instance_type" : "m1.small",
            "key_name": "PUBKEY",
            "vpc_security_group_ids" : ["${aws_security_group.default.id}"],
            "subnet_id": "${aws_subnet.default.id}",
            "tags" :  {
                "Name" : "web4",
                "InstancePlugin" : "terraform"
            },
            "connection" : {
                "user" : "ubuntu"
            },
            "provisioner" : {
                "remote_exec" : {
                    "inline" : [
                        "sudo apt-get -y update",
                        "sudo apt-get -y install nginx",
                        "sudo service nginx start"
                    ]
                }
            }
        }
    },
    "Tags" : {
        "other" : "values",
        "to" : "merge",
        "with" : "tags"
    },
    "Init" : "init string"
}

*/
type TFormat struct {

	// Resource : resource_type : name : map[string]interface{}
	Resource map[string]map[string]map[string]interface{} `json:"resource"`
}

// SpecPropertiesFormat is the schema in the Properties field of the instance.Spec JSON
type SpecPropertiesFormat struct {
	Type  string                 `json:"type"`
	Value map[string]interface{} `json:"value"`
}

// Validate performs local validation on a provision request.
func (p *plugin) Validate(req json.RawMessage) error {
	log.Debugln("validate", string(req))

	parsed := SpecPropertiesFormat{}
	err := json.Unmarshal([]byte(req), &parsed)
	if err != nil {
		return err
	}

	if parsed.Type == "" {
		return fmt.Errorf("no-resource-type:%s", string(req))
	}

	if len(parsed.Value) == 0 {
		return fmt.Errorf("no-value:%s", string(req))
	}
	return nil
}

func addUserData(m map[string]interface{}, key string, init string) {
	if v, has := m[key]; has {
		m[key] = fmt.Sprintf("%s\n%s", v, init)
	} else {
		m[key] = init
	}
}

func (p *plugin) terraformApply() error {
	p.applyLock.Lock()
	defer p.applyLock.Unlock()

	if p.applying {
		return nil
	}

	go func() {
		for {
			if err := p.lock.TryLock(); err == nil {
				defer p.lock.Unlock()
				p.doTerraformApply()
			}
			log.Debugln("Can't acquire lock, waiting")
			time.Sleep(time.Duration(int64(rand.NormFloat64())%1000) * time.Millisecond)
		}
	}()
	p.applying = true
	return nil
}

func (p *plugin) doTerraformApply() error {
	log.Infoln("Applying plan")
	cmd := exec.Command("terraform", "apply")
	cmd.Dir = p.Dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	output := io.MultiReader(stdout, stderr)
	go func() {
		reader := bufio.NewReader(output)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			log.WithField("terraform", "apply").Infoln(line)
		}
	}()
	return cmd.Run() // blocks
}

func (p *plugin) terraformShow() (map[string]interface{}, error) {
	re := regexp.MustCompile("(^instance-[0-9]+)(.tf.json)")

	result := map[string]interface{}{}

	fs := &afero.Afero{Fs: p.fs}
	// just scan the directory for the instance-*.tf.json files
	err := fs.Walk(p.Dir, func(path string, info os.FileInfo, err error) error {
		matches := re.FindStringSubmatch(info.Name())

		if len(matches) == 3 {
			id := matches[1]
			parse := map[string]interface{}{}

			buff, err := ioutil.ReadFile(filepath.Join(p.Dir, info.Name()))

			if err != nil {
				log.Warningln("Cannot parse:", err)
				return err
			}

			err = json.Unmarshal(buff, &parse)
			if err != nil {
				return err
			}

			if res, has := parse["resource"].(map[string]interface{}); has {
				var first map[string]interface{}
			res:
				for _, r := range res {
					if f, ok := r.(map[string]interface{}); ok {
						first = f
						break res
					}
				}
				if props, has := first[id]; has {
					result[id] = props
				}
			}
		}
		return nil
	})
	return result, err
}

func (p *plugin) parseTfStateFile() (map[string]interface{}, error) {
	// open the terraform.tfstate file
	buff, err := ioutil.ReadFile(filepath.Join(p.Dir, "terraform.tfstate"))
	if err != nil {

		// The tfstate file is not present this means we have to apply it first.
		if os.IsNotExist(err) {
			if err = p.terraformApply(); err != nil {
				return nil, err
			}
			return p.terraformShow()
		}
		return nil, err
	}

	// tfstate is a JSON so query it
	parsed := map[string]interface{}{}
	err = json.Unmarshal(buff, &parsed)
	if err != nil {
		return nil, err
	}

	if m1, has := parsed["modules"].([]interface{}); has && len(m1) > 0 {
		module := m1[0]
		if mm, ok := module.(map[string]interface{}); ok {
			if resources, ok := mm["resources"].(map[string]interface{}); ok {

				// the attributes are wrapped under each resource objects'
				// primary.attributes
				result := map[string]interface{}{}
				for k, rr := range resources {
					if r, ok := rr.(map[string]interface{}); ok {
						if primary, ok := r["primary"].(map[string]interface{}); ok {
							if attributes, ok := primary["attributes"]; ok {
								result[k] = attributes
							}
						}
					}
				}
				return result, nil
			}
		}
	}
	return nil, nil
}

func (p *plugin) ensureUniqueFile() string {
	for {
		if err := p.lock.TryLock(); err == nil {
			defer p.lock.Unlock()
			return ensureUniqueFile(p.Dir)
		}
		log.Infoln("Can't acquire lock, waiting")
		time.Sleep(time.Duration(int64(rand.NormFloat64())%1000) * time.Millisecond)
	}
}

func ensureUniqueFile(dir string) string {
	n := fmt.Sprintf("instance-%d", time.Now().Unix())
	// if we can open then we have to try again...  the file cannot exist currently
	if f, err := os.Open(filepath.Join(dir, n) + ".tf.json"); err == nil {
		f.Close()
		return ensureUniqueFile(dir)
	}
	return n
}

// Provision creates a new instance based on the spec.
func (p *plugin) Provision(spec instance.Spec) (*instance.ID, error) {
	// Simply writes a file and call terraform apply

	if spec.Properties == nil {
		return nil, fmt.Errorf("no-properties")
	}

	properties := SpecPropertiesFormat{}
	err := json.Unmarshal(*spec.Properties, &properties)
	if err != nil {
		return nil, err
	}

	// use timestamp as instance id
	name := p.ensureUniqueFile()

	id := instance.ID(name)

	// set the tags.
	// add a name
	if spec.Tags != nil {
		if _, has := spec.Tags["Name"]; !has {
			spec.Tags["Name"] = string(id)
		}
	}
	switch properties.Type {
	case "aws_instance", "azurerm_virtual_machine", "digitalocean_droplet", "google_compute_instance":
		if t, exists := properties.Value["tags"]; !exists {
			properties.Value["tags"] = spec.Tags
		} else if mm, ok := t.(map[string]interface{}); ok {
			// merge tags
			for tt, vv := range spec.Tags {
				mm[tt] = vv
			}
		}
	}

	// Use tag to store the logical id
	if spec.LogicalID != nil {
		if m, ok := properties.Value["tags"].(map[string]string); ok {
			m["LogicalID"] = string(*spec.LogicalID)
		}
	}

	// merge the inits
	switch properties.Type {
	case "aws_instance", "digitalocean_droplet":
		addUserData(properties.Value, "user_data", spec.Init)
	case "azurerm_virtual_machine":
		// os_profile.custom_data
		if m, has := properties.Value["os_profile"]; !has {
			properties.Value["os_profile"] = map[string]interface{}{
				"custom_data": spec.Init,
			}
		} else if mm, ok := m.(map[string]interface{}); ok {
			addUserData(mm, "custom_data", spec.Init)
		}
	case "google_compute_instance":
		// metadata_startup_script
		addUserData(properties.Value, "metadata_startup_script", spec.Init)
	}

	tfFile := TFormat{
		Resource: map[string]map[string]map[string]interface{}{
			properties.Type: {
				name: properties.Value,
			},
		},
	}

	buff, err := json.MarshalIndent(tfFile, "  ", "  ")
	log.Debugln("provision", id, "data=", string(buff), "err=", err)
	if err != nil {
		return nil, err
	}

	err = afero.WriteFile(p.fs, filepath.Join(p.Dir, string(id)+".tf.json"), buff, 0644)
	if err != nil {
		return nil, err
	}

	return &id, p.terraformApply()
}

// Destroy terminates an existing instance.
func (p *plugin) Destroy(instance instance.ID) error {
	fp := filepath.Join(p.Dir, string(instance)+".tf.json")
	log.Debugln("destroy", fp)
	err := p.fs.Remove(fp)
	if err != nil {
		return err
	}
	return p.terraformApply()
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (p *plugin) DescribeInstances(tags map[string]string) ([]instance.Description, error) {
	log.Debugln("describe-instances", tags)

	show, err := p.terraformShow()
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile("(.*)(instance-[0-9]+)")
	result := []instance.Description{}
	// now we scan for <instance_type.instance-<timestamp> as keys
scan:
	for k, v := range show {
		matches := re.FindStringSubmatch(k)
		if len(matches) == 3 {
			id := matches[2]

			inst := instance.Description{
				Tags:      terraformTags(v, "tags"),
				ID:        instance.ID(id),
				LogicalID: terraformLogicalID(v),
			}
			if len(tags) == 0 {
				result = append(result, inst)
			} else {
				for k, v := range tags {
					if inst.Tags[k] != v {
						continue scan // we implement AND
					}
				}
				result = append(result, inst)
			}
		}
	}
	return result, nil
}

func terraformTags(v interface{}, key string) map[string]string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	tags := map[string]string{}
	if mm, ok := m[key].(map[string]interface{}); ok {
		for k, v := range mm {
			tags[k] = fmt.Sprintf("%v", v)
		}
		return tags
	}
	for k, v := range m {
		if k != "tags.%" && strings.Index(k, "tags.") == 0 {
			n := k[len("tags."):]
			tags[n] = fmt.Sprintf("%v", v)
		}
	}
	return tags
}
func terraformLogicalID(v interface{}) *instance.LogicalID {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	v, exists := m["tags.LogicalID"]
	if exists {
		id := instance.LogicalID(fmt.Sprintf("%v", v))
		return &id
	}
	return nil
}

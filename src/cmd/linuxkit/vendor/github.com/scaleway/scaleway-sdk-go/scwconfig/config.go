package scwconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/scaleway/scaleway-sdk-go/logger"
	"github.com/scaleway/scaleway-sdk-go/utils"
	"gopkg.in/yaml.v2"
)

// Environment variables
const (
	// Up-to-date
	scwConfigPathEnv       = "SCW_CONFIG_PATH"
	scwAccessKeyEnv        = "SCW_ACCESS_KEY"
	scwSecretKeyEnv        = "SCW_SECRET_KEY"
	scwActiveProfileEnv    = "SCW_PROFILE"
	scwAPIURLEnv           = "SCW_API_URL"
	scwInsecureEnv         = "SCW_INSECURE"
	scwDefaultProjectIDEnv = "SCW_DEFAULT_PROJECT_ID"
	scwDefaultRegionEnv    = "SCW_DEFAULT_REGION"
	scwDefaultZoneEnv      = "SCW_DEFAULT_ZONE"

	// All deprecated (cli&terraform)
	terraformAccessKeyEnv    = "SCALEWAY_ACCESS_KEY" // used both as access key and secret key
	terraformSecretKeyEnv    = "SCALEWAY_TOKEN"
	terraformOrganizationEnv = "SCALEWAY_ORGANIZATION"
	terraformRegionEnv       = "SCALEWAY_REGION"
	cliTLSVerifyEnv          = "SCW_TLSVERIFY"
	cliOrganizationEnv       = "SCW_ORGANIZATION"
	cliRegionEnv             = "SCW_REGION"
	cliSecretKeyEnv          = "SCW_TOKEN"

	// TBD
	//cliVerboseEnv         = "SCW_VERBOSE_API"
	//cliDebugEnv           = "DEBUG"
	//cliNoCheckVersionEnv  = "SCW_NOCHECKVERSION"
	//cliTestWithRealAPIEnv = "TEST_WITH_REAL_API"
	//cliSecureExecEnv      = "SCW_SECURE_EXEC"
	//cliGatewayEnv         = "SCW_GATEWAY"
	//cliSensitiveEnv       = "SCW_SENSITIVE"
	//cliAccountAPIEnv      = "SCW_ACCOUNT_API"
	//cliMetadataAPIEnv     = "SCW_METADATA_API"
	//cliMarketPlaceAPIEnv  = "SCW_MARKETPLACE_API"
	//cliComputePar1APIEnv  = "SCW_COMPUTE_PAR1_API"
	//cliComputeAms1APIEnv  = "SCW_COMPUTE_AMS1_API"
	//cliCommercialTypeEnv  = "SCW_COMMERCIAL_TYPE"
	//cliTargetArchEnv      = "SCW_TARGET_ARCH"
)

// Config interface is made of getters to retrieve
// the config field by field.
type Config interface {
	GetAccessKey() (accessKey string, exist bool)
	GetSecretKey() (secretKey string, exist bool)
	GetAPIURL() (apiURL string, exist bool)
	GetInsecure() (insecure bool, exist bool)
	GetDefaultProjectID() (defaultProjectID string, exist bool)
	GetDefaultRegion() (defaultRegion utils.Region, exist bool)
	GetDefaultZone() (defaultZone utils.Zone, exist bool)
}

type configV2 struct {
	profile       `yaml:",inline"`
	ActiveProfile *string             `yaml:"active_profile,omitempty"`
	Profiles      map[string]*profile `yaml:"profiles,omitempty"`

	// withProfile is used by LoadWithProfile to handle the following priority order:
	// c.withProfile > os.Getenv("SCW_PROFILE") > c.ActiveProfile
	withProfile string
}

type profile struct {
	AccessKey        *string `yaml:"access_key,omitempty"`
	SecretKey        *string `yaml:"secret_key,omitempty"`
	APIURL           *string `yaml:"api_url,omitempty"`
	Insecure         *bool   `yaml:"insecure,omitempty"`
	DefaultProjectID *string `yaml:"default_project_id,omitempty"`
	DefaultRegion    *string `yaml:"default_region,omitempty"`
	DefaultZone      *string `yaml:"default_zone,omitempty"`
}

func unmarshalConfV2(content []byte) (*configV2, error) {
	var config configV2

	err := yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *configV2) catchInvalidProfile() (*configV2, error) {
	activeProfile, err := c.getActiveProfile()
	if err != nil {
		return nil, err
	}
	if activeProfile == "" {
		return c, nil
	}

	_, exist := c.Profiles[activeProfile]
	if !exist {
		return nil, fmt.Errorf("profile %s does not exist %s", activeProfile, inConfigFile())
	}
	return c, nil
}

func (c *configV2) getActiveProfile() (string, error) {
	switch {
	case c.withProfile != "":
		return c.withProfile, nil
	case os.Getenv(scwActiveProfileEnv) != "":
		return os.Getenv(scwActiveProfileEnv), nil
	case c.ActiveProfile != nil:
		if *c.ActiveProfile == "" {
			return "", fmt.Errorf("active_profile key cannot be empty %s", inConfigFile())
		}
		return *c.ActiveProfile, nil
	default:
		return "", nil
	}
}

// GetAccessKey retrieve the access key from the config.
// It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetAccessKey() (string, bool) {
	envValue, _, envExist := getenv(scwAccessKeyEnv, terraformAccessKeyEnv)
	activeProfile, _ := c.getActiveProfile()

	var accessKey string
	switch {
	case envExist:
		accessKey = envValue
	case activeProfile != "" && c.Profiles[activeProfile].AccessKey != nil:
		accessKey = *c.Profiles[activeProfile].AccessKey
	case c.AccessKey != nil:
		accessKey = *c.AccessKey
	default:
		logger.Warningf("no access key found")
		return "", false
	}

	if accessKey == "" {
		logger.Warningf("access key is empty")
	}

	return accessKey, true
}

// GetSecretKey retrieve the secret key from the config.
// It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetSecretKey() (string, bool) {
	envValue, _, envExist := getenv(scwSecretKeyEnv, cliSecretKeyEnv, terraformSecretKeyEnv, terraformAccessKeyEnv)
	activeProfile, _ := c.getActiveProfile()

	var secretKey string
	switch {
	case envExist:
		secretKey = envValue
	case activeProfile != "" && c.Profiles[activeProfile].SecretKey != nil:
		secretKey = *c.Profiles[activeProfile].SecretKey
	case c.SecretKey != nil:
		secretKey = *c.SecretKey
	default:
		logger.Warningf("no secret key found")
		return "", false
	}

	if secretKey == "" {
		logger.Warningf("secret key is empty")
	}

	return secretKey, true
}

// GetAPIURL retrieve the api url from the config.
// It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetAPIURL() (string, bool) {
	envValue, _, envExist := getenv(scwAPIURLEnv)
	activeProfile, _ := c.getActiveProfile()

	var apiURL string
	switch {
	case envExist:
		apiURL = envValue
	case activeProfile != "" && c.Profiles[activeProfile].APIURL != nil:
		apiURL = *c.Profiles[activeProfile].APIURL
	case c.APIURL != nil:
		apiURL = *c.APIURL
	default:
		return "", false
	}

	if apiURL == "" {
		logger.Warningf("api URL is empty")
	}

	return apiURL, true
}

// GetInsecure retrieve the insecure flag from the config.
// It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetInsecure() (bool, bool) {
	envValue, envKey, envExist := getenv(scwInsecureEnv, cliTLSVerifyEnv)
	activeProfile, _ := c.getActiveProfile()

	var insecure bool
	var err error
	switch {
	case envExist:
		insecure, err = strconv.ParseBool(envValue)
		if err != nil {
			logger.Warningf("env variable %s cannot be parsed: %s is invalid boolean ", envKey, envValue)
			return false, false
		}

		if envKey == cliTLSVerifyEnv {
			insecure = !insecure // TLSVerify is the inverse of Insecure
		}
	case activeProfile != "" && c.Profiles[activeProfile].Insecure != nil:
		insecure = *c.Profiles[activeProfile].Insecure
	case c.Insecure != nil:
		insecure = *c.Insecure
	default:
		return false, false
	}

	return insecure, true
}

// GetDefaultProjectID retrieve the default project ID
// from the config. Legacy configs used the name
// "organization ID" or "organization" for
// this field. It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetDefaultProjectID() (string, bool) {
	envValue, _, envExist := getenv(scwDefaultProjectIDEnv, cliOrganizationEnv, terraformOrganizationEnv)
	activeProfile, _ := c.getActiveProfile()

	var defaultProj string
	switch {
	case envExist:
		defaultProj = envValue
	case activeProfile != "" && c.Profiles[activeProfile].DefaultProjectID != nil:
		defaultProj = *c.Profiles[activeProfile].DefaultProjectID
	case c.DefaultProjectID != nil:
		defaultProj = *c.DefaultProjectID
	default:
		return "", false
	}

	// todo: validate format
	if defaultProj == "" {
		logger.Warningf("default project ID is empty")
	}

	return defaultProj, true
}

// GetDefaultRegion retrieve the default region
// from the config. It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetDefaultRegion() (utils.Region, bool) {
	envValue, _, envExist := getenv(scwDefaultRegionEnv, cliRegionEnv, terraformRegionEnv)
	activeProfile, _ := c.getActiveProfile()

	var defaultRegion string
	switch {
	case envExist:
		defaultRegion = v1RegionToV2(envValue)
	case activeProfile != "" && c.Profiles[activeProfile].DefaultRegion != nil:
		defaultRegion = *c.Profiles[activeProfile].DefaultRegion
	case c.DefaultRegion != nil:
		defaultRegion = *c.DefaultRegion
	default:
		return "", false
	}

	// todo: validate format
	if defaultRegion == "" {
		logger.Warningf("default region is empty")
	}

	return utils.Region(defaultRegion), true
}

// GetDefaultZone retrieve the default zone
// from the config. It will check the following order:
// env, legacy env, active profile, default profile
//
// If the config is present in one of the above environment the
// value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will
// be false.
func (c *configV2) GetDefaultZone() (utils.Zone, bool) {
	envValue, _, envExist := getenv(scwDefaultZoneEnv)
	activeProfile, _ := c.getActiveProfile()

	var defaultZone string
	switch {
	case envExist:
		defaultZone = envValue
	case activeProfile != "" && c.Profiles[activeProfile].DefaultZone != nil:
		defaultZone = *c.Profiles[activeProfile].DefaultZone
	case c.DefaultZone != nil:
		defaultZone = *c.DefaultZone
	default:
		return "", false
	}

	// todo: validate format
	if defaultZone == "" {
		logger.Warningf("default zone is empty")
	}

	return utils.Zone(defaultZone), true
}

func getenv(upToDateKey string, deprecatedKeys ...string) (string, string, bool) {
	value, exist := os.LookupEnv(upToDateKey)
	if exist {
		logger.Infof("reading value from %s", upToDateKey)
		return value, upToDateKey, true
	}

	for _, key := range deprecatedKeys {
		value, exist := os.LookupEnv(key)
		if exist {
			logger.Infof("reading value from %s", key)
			logger.Warningf("%s is deprecated, please use %s instead", key, upToDateKey)
			return value, key, true
		}
	}

	return "", "", false
}

const (
	v1RegionFrPar = "par1"
	v1RegionNlAms = "ams1"
)

// configV1 is a Scaleway CLI configuration file
type configV1 struct {
	// Organization is the identifier of the Scaleway organization
	Organization string `json:"organization"`

	// Token is the authentication token for the Scaleway organization
	Token string `json:"token"`

	// Version is the actual version of scw CLI
	Version string `json:"version"`
}

func unmarshalConfV1(content []byte) (*configV1, error) {
	var config configV1
	err := json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	return &config, err
}

func (v1 *configV1) toV2() *configV2 {
	return &configV2{
		profile: profile{
			DefaultProjectID: &v1.Organization,
			SecretKey:        &v1.Token,
			// ignore v1 version
		},
	}
}

func v1RegionToV2(region string) string {
	switch region {
	case v1RegionFrPar:
		logger.Warningf("par1 is a deprecated name for region, use fr-par instead")
		return "fr-par"
	case v1RegionNlAms:
		logger.Warningf("ams1 is a deprecated name for region, use nl-ams instead")
		return "nl-ams"
	default:
		return region
	}
}

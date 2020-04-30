package scw

import (
	"os"
	"strconv"

	"github.com/scaleway/scaleway-sdk-go/logger"
)

// Environment variables
const (
	// Up-to-date
	scwCacheDirEnv              = "SCW_CACHE_DIR"
	scwConfigPathEnv            = "SCW_CONFIG_PATH"
	scwAccessKeyEnv             = "SCW_ACCESS_KEY"
	scwSecretKeyEnv             = "SCW_SECRET_KEY" // #nosec G101
	scwActiveProfileEnv         = "SCW_PROFILE"
	scwAPIURLEnv                = "SCW_API_URL"
	scwInsecureEnv              = "SCW_INSECURE"
	scwDefaultOrganizationIDEnv = "SCW_DEFAULT_ORGANIZATION_ID"
	scwDefaultRegionEnv         = "SCW_DEFAULT_REGION"
	scwDefaultZoneEnv           = "SCW_DEFAULT_ZONE"

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

const (
	v1RegionFrPar = "par1"
	v1RegionNlAms = "ams1"
)

func LoadEnvProfile() *Profile {
	p := &Profile{}

	accessKey, _, envExist := getEnv(scwAccessKeyEnv, terraformAccessKeyEnv)
	if envExist {
		p.AccessKey = &accessKey
	}

	secretKey, _, envExist := getEnv(scwSecretKeyEnv, cliSecretKeyEnv, terraformSecretKeyEnv, terraformAccessKeyEnv)
	if envExist {
		p.SecretKey = &secretKey
	}

	apiURL, _, envExist := getEnv(scwAPIURLEnv)
	if envExist {
		p.APIURL = &apiURL
	}

	insecureValue, envKey, envExist := getEnv(scwInsecureEnv, cliTLSVerifyEnv)
	if envExist {
		insecure, err := strconv.ParseBool(insecureValue)
		if err != nil {
			logger.Warningf("env variable %s cannot be parsed: %s is invalid boolean", envKey, insecureValue)
		}

		if envKey == cliTLSVerifyEnv {
			insecure = !insecure // TLSVerify is the inverse of Insecure
		}

		p.Insecure = &insecure
	}

	organizationID, _, envExist := getEnv(scwDefaultOrganizationIDEnv, cliOrganizationEnv, terraformOrganizationEnv)
	if envExist {
		p.DefaultOrganizationID = &organizationID
	}

	region, _, envExist := getEnv(scwDefaultRegionEnv, cliRegionEnv, terraformRegionEnv)
	if envExist {
		region = v1RegionToV2(region)
		p.DefaultRegion = &region
	}

	zone, _, envExist := getEnv(scwDefaultZoneEnv)
	if envExist {
		p.DefaultZone = &zone
	}

	return p
}

func getEnv(upToDateKey string, deprecatedKeys ...string) (string, string, bool) {
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

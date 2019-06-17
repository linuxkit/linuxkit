# Scaleway config

## TL;DR

Recommended config file:

```yaml
# get your credentials on https://console.scaleway.com/account/credentials
access_key: SCWXXXXXXXXXXXXXXXXX
secret_key: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
default_project_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
default_region: fr-par
default_zone: fr-par-1
```

## Config file path

This package will try to locate the config file in the following ways:

1. Custom directory: `$SCW_CONFIG_PATH`
2. [XDG base directory](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html): `$XDG_CONFIG_HOME/scw/config.yaml`
3. Home directory: `$HOME/.config/scw/config.yaml` (`%USERPROFILE%/.config/scw/config.yaml` on windows)

## V1 config (DEPRECATED)

The V1 config `.scwrc` is supported but deprecated.
When found in the home directory, the V1 config is automatically migrated to a V2 config file in `$HOME/.config/scw/config.yaml`.

## Reading config order

When getting the value of a config field, the following priority order will be respected:

1. Environment variable
2. Legacy environment variable
3. Config file V2
4. Config file V1

## Environment variables

| Variable                  | Description                                                                                                                             | Legacy variables                                                                                              |
| :------------------------ | :-------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------ |
| `$SCW_ACCESS_KEY`         | Access key of a token ([get yours](https://console.scaleway.com/account/credentials))                                                   | `$SCALEWAY_ACCESS_KEY` (used by terraform)                                                                    |
| `$SCW_SECRET_KEY`         | Secret key of a token ([get yours](https://console.scaleway.com/account/credentials))                                                   | `$SCW_TOKEN` (used by cli), `$SCALEWAY_TOKEN` (used by terraform), `$SCALEWAY_ACCESS_KEY` (used by terraform) |
| `$SCW_DEFAULT_PROJECT_ID` | Your default project ID, if you don't have one use your organization ID ([get yours](https://console.scaleway.com/account/credentials)) | `$SCW_ORGANIZATION` (used by cli),`$SCALEWAY_ORGANIZATION` (used by terraform)                                |
| `$SCW_DEFAULT_REGION`     | Your default [region](https://developers.scaleway.com/en/quickstart/#region-and-zone)                                                   | `$SCW_REGION` (used by cli),`$SCALEWAY_REGION` (used by terraform)                                            |
| `$SCW_DEFAULT_ZONE`       | Your default [availability zone](https://developers.scaleway.com/en/quickstart/#region-and-zone)                                        | `$SCW_ZONE` (used by cli),`$SCALEWAY_ZONE` (used by terraform)                                                |
| `$SCW_API_URL`            | Url of the API                                                                                                                          | -                                                                                                             |
| `$SCW_INSECURE`           | Set this to `true` to enable the insecure mode                                                                                          | `$SCW_TLSVERIFY` (inverse flag used by the cli)                                                               |

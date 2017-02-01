# Build Azure VHD
To create the Azure VHD, the following will be needed:
* An azure account
* A Standard Storage account
* A container (bucket) in the above storage account (private)
* The access key associated with the above storage account
* (opt) the url for the docker version you want to use in the VHD

In your terminal, with docker installed, run the following:

```
export AZURE_STG_ACCOUNT_NAME="<your-storage-account>"
export AZURE_STG_ACCOUNT_KEY="<your-access-key>"
export CONTAINER_NAME="<a-bucket-name>"
make uploadvhd DOCKER_BIN_URL="<tgz-docker-url>"
```

The above will output a URL which you can then use to deploy on editions.
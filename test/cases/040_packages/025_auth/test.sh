#!/bin/sh
# SUMMARY: Check that we can access a registry with auth
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	docker kill "${REGISTRY_NAME}" || true
	DOCKER_CONFIG="${DOCKER_CONFIG}" docker buildx rm "${BUILDKIT_NAME}" || true
	[ -n "${CACHDIR}" ] && rm -rf "${CACHDIR}"
	[ -n "${DOCKER_CONFIG}" ] && rm -rf "${DOCKER_CONFIG}"
	[ -n "${REGISTRY_DIR}" ] && rm -rf "${REGISTRY_DIR}"
}
trap clean_up EXIT

# determine platform
ARCH=$(uname -m)
if [ "${ARCH}" = "x86_64" ]; then
  ARCH="amd64"
elif [ "${ARCH}" = "aarch64" ]; then
  ARCH="arm64"
fi
PLATFORM="linux/${ARCH}"


# container names
REGISTRY_NAME="test-registry-$$"
BUILDKIT_NAME="test-buildkitd-$$"

# start a registry with auth
REGISTRY_USER="testuser"
REGISTRY_PASS="testpass"
REGISTRY_PORT="5000"
REGISTRY_DIR=$(mktemp -d)
mkdir -p "$REGISTRY_DIR/auth"
docker run --rm \
  --entrypoint htpasswd \
  httpd:2 -Bbn "${REGISTRY_USER}" "${REGISTRY_PASS}" > "$REGISTRY_DIR/auth/htpasswd"

# Start registry
REGISTRY_CID=$(docker run -d --rm \
  -p ":${REGISTRY_PORT}" \
  -v "$REGISTRY_DIR/auth:/auth" \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
  --name "${REGISTRY_NAME}" \
  registry:3)

REGISTRY_IP=$(docker inspect "${REGISTRY_NAME}" \
  --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')

IMAGENAME="${REGISTRY_IP}:${REGISTRY_PORT}/myimage"

# start an insecure buildkit so we can load an image to the registry
cat > buildkitd.toml <<EOF
[registry."${REGISTRY_IP}:${REGISTRY_PORT}"]
  insecure = true
  http = true
EOF

# save the credentials
credsb64=$(printf "%s" "${REGISTRY_USER}:${REGISTRY_PASS}" | base64)

# DO NOT export DOCKER_CONFIG, as that will cause the thing we are testing to succeed.
# we need to be explicit about it.
DOCKER_CONFIG=$(pwd)/docker-config
rm -rf "${DOCKER_CONFIG}"
mkdir -p "${DOCKER_CONFIG}"
cat > "${DOCKER_CONFIG}/config.json" <<EOF
{
  "auths": {
    "${REGISTRY_IP}:5000": {
      "auth": "${credsb64}"
    }
  }
}
EOF

DOCKER_CONFIG=${DOCKER_CONFIG} docker buildx create \
  --name "${BUILDKIT_NAME}" \
  --driver docker-container \
  --buildkitd-config "$(pwd)/buildkitd.toml" \
  --bootstrap

DOCKER_CONFIG=${DOCKER_CONFIG} docker buildx build \
  --builder "${BUILDKIT_NAME}" \
  --file Dockerfile.base \
  --tag "${IMAGENAME}" \
  --push \
  --progress plain \
  --platform "${PLATFORM}" \
  .

# Generate Dockerfile for pkg with FROM
cat > Dockerfile <<EOF
FROM "${IMAGENAME}"
RUN echo SUCCESS
EOF


CACHEDIR=$(mktemp -d)

# 3 tests:
# 1. build a package with no auth - should fail
# 2. build a package with explicit auth - should succeed
# 3. build a package with auth in the config - should succeed
if linuxkit --cache "${CACHEDIR}" pkg build --platforms "${PLATFORM}" \
  --builder-config "$(pwd)/buildkitd.toml" --force \
  .; then 
  echo "Test 1 failed: build succeeded without auth"
  exit 1
fi

linuxkit --cache "${CACHEDIR}" pkg build --platforms "${PLATFORM}" \
  --builder-config "$(pwd)/buildkitd.toml" --force \
  --registry-creds "${REGISTRY_IP}:${REGISTRY_PORT}=${REGISTRY_USER}:${REGISTRY_PASS}" \
  .

DOCKER_CONFIG=${DOCKER_CONFIG} linuxkit --cache "${CACHEDIR}" pkg build --platforms "${PLATFORM}" \
  --builder-config "$(pwd)/buildkitd.toml" --force \
  .


exit 0

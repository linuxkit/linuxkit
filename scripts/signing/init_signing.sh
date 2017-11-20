# USAGE: ./init_signing.sh linuxkit/repo

if [[ -z  $DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE  ]]
then
    echo "must set DOCKER_CONTENT_TRUST_ROOT_PASSPHRASE"
    exit 1
fi

if [[ -z  $DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE  ]]
then
    echo "must set DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"
    exit 1
fi

docker trust signer add justin $1 --key justin.crt

docker trust signer add rolf $1 --key rolf.crt

docker trust signer add ian $1 --key ian.crt --key ian_arm.crt

docker trust signer add avi $1 --key avi.crt --key avi_arm.crt

docker trust signer add riyaz $1 --key riyaz.crt

echo "Successfully set up signing for $1"

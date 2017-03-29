# AWS support

The aim is provide good integration of Moby with Amazon AWS.

Currently there is a container ([cli](cli/)) containing the AWS tools to manage AWS images and a Alpine based image ([alpine-aws](alpine-aws/)) which contains the integration services for AWS.

## Roadmap

**Near-term:**
- Package AWS Integrations tools/cloudinit into a container image to be used in yaml files.
- Add support for [building AMIs](https://github.com/docker/moby/pull/1119)

**Mid-term:**
- Regular CI jobs testing AWS integration

Linuxkit Tests Labels
=====================

## Usage of Artifacts vs Temporary Directory

As the Build Machines have no secrets they will not be able to test running the build output on any cloud providers.
In this instance, the build tests should copy their output to the `LINUXKIT_ARTIFACTS_DIRECTORY`

## Labels

The `gcp` label is applied when the system where the tests are being run meets the requirements for using Google Cloud Platform.

These requirements are:
- The system has the necessary `CLOUDSDK_*` environment variables exported
- The system has either keys for a GCP service account or is able to use application default credentials

The `packet.net` label is applied when a system is able to create machines on Packet.net

The `vmware` label is used when the machine has VMware Workstation or Fusion installed

LinuxKit Testing and CI
=======================

## Testing

The test suite uses [`rtf`](https://github.com/linuxkit/rtf)
To install this you should use `make bin/rtf && make install`.

### Running the tests

To run the test suite:

```
cd test
rtf -x run
```

This will run the tests and put the results in a the `_results` directory!

Run control is handled using labels and with pattern matching.
To run add a label you may use:

```
rtf -x -l slow run
```

To run tests that match the pattern `linuxkit.examples` you would use the following command:

```
rtf -x run linuxkit.examples
```

### Writing new tests

To add a new test, you should first decide which group it should be added to.
Check the `test/cases` folder for a list of existing groups.
Groups can be nested and there examples of this in the existing set of test cases.

If you feel that a new group is warranted you can create one by `mkdir 000_name` where
`000` is the desired order and `name` is the name you would like to use
You must copy an existing `group.sh` in to this folder and adjust as required or you may use the
[example](https://github.com/linuxkit/rtf/tree/master/etc/templates/group.sh)

To write your test, create a folder within the group using the `000_name` format as described aboce.
You should then copy an existing `test.sh` in to this directory and amdend it,
or start from an [example](http://github.com/linuxkit/rtf/tree/master/etc/templates/test.sh)

If your test can only be run when certain conditions are met, you should consider adding a label to
avoid it being run by default and document the use of the label in `tests/README.md`

## Continuous Integration

The LiunxKit CI system uses [DatakitCI](https://github.com/moby/datakit/tree/master/ci)
The configuration can be found [here](https://github.com/linuxkit/linuxkit-ci)

## PR Testing

Each PR is tested on disposable VM's spawned in Google Cloud Platform
This machine has no privileges or credentials to talk to GCP or other cloud platforms.

TODO: Add instructions on how to build a base image for LinuxKit CI in GCP.

LinuxKit CI runs `make ci-pr` in the VM.
This target runs the tests using `rtf` and the results directory is SCP'd back to the controller.
The test results will be stored in DataKit for additional access
Additionally a the `./artifacts` folder is SCP'd back to the controller.

If the tests passed the next step is to check if the kernel config test image runs on GCP.
The `./artifacts/test.img.tar.gz` file is used to create a GCP image by the Python scripts
that are part of LiunxKit CI. It runs this image and greps for the success message in the logs.

## Branch and Tag testing

Branches and Tags are tested on a dedicated machine that runs in GCP.

LinuxKit CI runs `make ci` or `make ci-tag` in the VM
This target runs the tests using `rtf` and the results directory is SCP'd back to the controller.
The test results will be stored in DataKit.

If the tests pass, the GCP test is run in the same manner as described for PR tests.

Finally, CI will run `make test-ltp` which will run the Linux Testing Project tests.

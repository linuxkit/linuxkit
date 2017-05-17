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

You can list the tests which will be run using:

```
rtf list
```

Some tests may be marked as `SKIP` and `LABELS` column will typically provide and indication as to why a test may be skipped.

To run tests that match the pattern `linuxkit.build` you would use the following command:

```
rtf -x run linuxkit.build
```

### Writing new tests

To add a new test, you should first decide which group it should be added to.
Check the `test/cases` folder for a list of existing groups.
Groups can be nested and there examples of this in the existing set of test cases.

If you feel that a new group is warranted you can create one by `mkdir 000_name` where
`000` is the desired order and `name` is the name you would like to use
You must copy an existing `group.sh` in to this folder and adjust as required or you may use the
[example](https://github.com/linuxkit/rtf/tree/master/etc/templates/group.sh)

To write your test, create a folder within the group using the `000_name` format as described above.
You should then copy an existing `test.sh` in to this directory and amdend it,
or start from an [example](http://github.com/linuxkit/rtf/tree/master/etc/templates/test.sh)

If your test can only be run when certain conditions are met, you should consider adding a label to
avoid it being run by default and document the use of the label in `tests/README.md`

## Continuous Integration

*Note: This will hopefully change significantly soon*

The LinuxKit CI system uses [DatakitCI](https://github.com/moby/datakit/tree/master/ci)
The configuration can be found [here](https://github.com/linuxkit/linuxkit-ci)

The test logs are available via the `Details` link but also via this [website](https://linuxkit.datakit.ci). The raw test logs are also stored on [GitHub](https://github.com/linuxkit/linuxkit-logs) where each branch contains the results for a particular run.

## PR Testing

Each PR is tested on disposable VMs spawned in Google Cloud Platform
This machine has no privileges or credentials to talk to GCP or other cloud platforms.

TODO: Add instructions on how to build a base image for LinuxKit CI in GCP.

LinuxKit CI runs `make ci-pr` in the VM.
This target runs the tests using `rtf` and the results directory is `scp`'ed back to the controller.
The test results are stored in DataKit.

If the tests passed the next step is to run additional platform tests.
LinxKit CI runs `make ci-platforms` from the controller.
This will use `rtf` to run tests on all local and/or cloud platforms in the test suite.

## Branch and Tag testing

Branches and Tags are tested on a dedicated machine that runs in GCP.

LinuxKit CI runs `make ci` or `make ci-tag` in the VM
This target runs the tests using `rtf` and the results directory is SCP'd back to the controller.
The labels supplied to `rtf` via the `Makefile` will enable slower tests like the Linux Testing Project tests
for more thorough test coverage.
The test results will be stored in DataKit.

If the tests pass, the platform tests are run in the same manner as described for PR's. 

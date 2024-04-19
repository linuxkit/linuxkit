# testing --input-tar

This test works by building two tar files, and checking logs.
This only works because we use verbose logs.

The two files - `test1.yml` and `test2.yml` are identical, except for some changed lines.

The test script - `test.sh` - builds an image from `test1.yml`, then uses its output
as `--input-tar` for building from `test2.yml`. It then checks the output logs to make sure
that expected sections are copied over, and unexpected ones are not.

**Note:** If you make any changes to either test file, mark here and in `test.sh` so we know what has changed.

Changes:

- added one entry in `init`
- changed the command in `onboot[1]`
- removed `services[1]`, which causes `services[2]` to become `services[1]`, and thus should not be copied either, as order may matter.

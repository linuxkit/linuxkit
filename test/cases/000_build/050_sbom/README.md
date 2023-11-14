# SBoM Test

Test that SBoM gets generated and unified.
This test does not launch the image, so it doesn't matter much that what is in it is runnable,
only that it gets built.

This test uses local packages inside the directory, to ensure that we get a known and controlled
SBoM.

How it works:

1. Builds the packages in [./package1](./package1) and [./package2](./package2)
1. Builds the image in [./test.yml](./test.yml)
1. Checks that the image contains an SBoM in the expected location
1. Checks that the SBoM contains at least some expected packages

## To update

If you change the packages in [./package1](./package1) or [./package2](./package2), you will need
to update the [./test.yml](./test.yml) file to reflect the new versions.

1. `linuxkit pkg show-tag ./package1`
1. `linuxkit pkg show-tag ./package2`
1. Update the `onboot` section of [./test.yml](./test.yml) with the new versions

## Probational channel

The goal of the probational channel is to collect more than one (potentially
all) of the projects into one linuxkit image, so that users can use them in
tandem.

The basic idea is to collect all patches, kernel configuration, kernel command
line options, userspace binaries, etc. by parsing the individual yaml files for
the project, looking in the project's patches directory, and by using the newly
added kernel_config.probational file, that indicates the particular
configuration options needed for this project to work. Ultimately, the goal
would be to use the kernel-config project style configuration everywhere, so we
don't need any kernel_config-xxx files at all, anymore, and kernel_config can
be used for both probational and regular config for a project.

### Building probational

There are three steps:

    make # in probational/ project

This generates the probational.yml file in this directory, which is a
collection of all the configuration.

    make IMAGE=probational # in kernel-config/ project

This builds the probational kernel with all the specified options enabled.

    sed -i s,your-probational-image,linuxkit/probational:XXXX
    ../../moby build probational && ../../linuxkit run probational

These build the actual probational image (assuming you substitute XXXX with
your kernel build from step 2), and then run it.

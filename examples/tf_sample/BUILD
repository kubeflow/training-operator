###############################################################################################
# Build a tarball that can be used to install the example code on Cloud ML workers.
#
# This is a pretty simplistic approach to building a "pip" package.
#  1. We don't rewrite any imports.
load("//tools/build_defs/pkg:pkg.bzl", "pkg_tar")

pkg_tar(
    name = "sources",
    files = glob(
        [
            "*.py",
        ],
        exclude = [
            # We don't exclude the tests because we want to run the tests inside the container.
            # Setup needs to be installed in the root of the package.
            "setup.py",
        ],
    ),
    package_dir = "tf_sample",
)

pkg_tar(
    name = "setup",
    files =
        [
            "setup.py",
        ],
)

pkg_tar(
    name = "pip_pkg",
    deps = [
        ":setup",
        ":sources",
    ],
)

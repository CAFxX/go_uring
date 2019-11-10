# go_uring :ring:
experimental io_uring library for go

:warning: **PRE-ALPHA: THIS REPO IS FOR EXPERIMENTAL PURPOSES
ONLY, DO NOT USE FOR ANYTHING IMPORTANT** :warning:

## Requirements

### Build
- A working Go installation.
  - You need CGO to be enabled (it is by default).
  - Cross-compilation has not been tested, and may not work.
- You need to have [`liburing`](https://git.kernel.dk/cgit/liburing/) installed.
  - There is a bundled release of `liburing` in the repo. 
    You can install it by running `make && make install` from its
    subdirectory (you will need a working C compiler and `make`). 

### Running
- Linux >= 5.1
  - If your kernel does not support io_uring you will get a panic
    when trying to use the library.

## Usage
See [documentation](https://godoc.org/github.com/CAFxX/go_uring).

## TODO
Lots. The code is horrible, as it uses cgo.

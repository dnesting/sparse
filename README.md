# github.com/dnesting/sparse

[![PkgGoDev](https://pkg.go.dev/badge/github.com/dnesting/sparse)](https://pkg.go.dev/github.com/dnesting/sparse)

This is a work in progress.

Sparse is an approach to reading and seeking through sparse input.
Types are defined for reading from streaming sparse input and random
access input.  A Buffer type implements these interfaces, allowing
arbitrary amounts of data to be stored at random sparse "locations"
in a pseudo-file.

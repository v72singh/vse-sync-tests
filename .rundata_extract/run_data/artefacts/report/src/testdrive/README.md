# testdrive

`testdrive` is a library for:

 * Generating [Asciidoc][1] test results for human consumption


## testdrive.asciidoc

Module `testdrive.asciidoc` can be used to generate [Asciidoc][1] test results
from lines of JSON (one line per test case result):

    $ python3 -m testdrive.run https://github.com/redhat-partner-solutions/testdrive/ examples/sequence/tests.json | \
      python3 -m testdrive.asciidoc "examples.sequence" - | tee results.adoc
    === Test Suite: examples.sequence

    ==== Summary

    [cols=2*.^a]
    |===


    |
    *hostname*
    |
    _not known_

    |
    *started*
    |
    2023-07-31T13:29:08.844977+00:00
    ...

To include this in a simple report:

    $ cat report.adoc
    = Test Report

    :toc:

    == Test Results

    <<<
    include::results.adoc[]

    $ asciidoctor -a toc report.adoc && firefox report.html

[1]: https://docs.asciidoctor.org/asciidoc/latest/

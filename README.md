enmime [![Build Status](https://travis-ci.org/jhillyerd/go.enmime.png?branch=master)](https://travis-ci.org/jhillyerd/go.enmime) [![GoDoc](https://godoc.org/github.com/jhillyerd/go.enmime?status.png)](https://godoc.org/github.com/jhillyerd/go.enmime)
======

enmime is a MIME parsing library for Go.  It's built ontop of Go's included mime/multipart
support, but is geared towards parsing MIME encoded emails.

It is being developed in tandem with the Inbucket email service.

API documentation can be found here:
http://godoc.org/github.com/jhillyerd/go.enmime

Development Status
------------------
enmime is alpha quality: it works but has not been tested with a wide variety of source data,
and it's likely the API will evolve some before an official release.

About
-----
enmime is written in [Google Go][1].

enmime is open source software released under the MIT License.  The latest
version can be found at https://github.com/jhillyerd/go.enmime

[1]: http://golang.org/

Forked Version [![Build Status](https://travis-ci.org/cention-sany/go.enmime.png?branch=master)](https://travis-ci.org/cention-sany/go.enmime)
--------------
enmime is forked to use forked stdlib mime, multipart, quotedprintable and textproto packages in order to parse some bad email format. This will increase chances to parse emails successfully.

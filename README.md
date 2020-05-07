# go-gemini

go-gemini is a library that provides an easy interface to create client and servers that speak the [Gemini protocol](https://gemini.circumlunar.space/).

This library was forked from [~yotam/go-gemini](https://git.sr.ht/~yotam/go-gemini/) to add additional features, as well as update it to support spec v0.11.0. At the time of forking, it had not seen any new commit for 5 months, and was based on v0.9.2. If there are any future upstream updates, I will make an effort to include them.

## Features

## Example Server
The repository comes with an example server that respond with an hardcoded text
to the root page. To build the server run the following command:

    make build

## License
The GNU Lesser General Public License or LGPL, version 3.0. See [LICENSE](LICENSE) for details. The gist of it is that proprietary code **can** legally make use of this library unaltered, but any modifications to the library source code must be released, under the same license.

The original library was under the ISC License, see [OLD-LICENSE](OLD-LICENSE) for details. The ISC is permissive and GPL-compatible, so relicensing the library is legally allowed.

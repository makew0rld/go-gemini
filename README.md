# go-gemini

go-gemini is a library that provides an easy interface to create client and servers that speak the [Gemini protocol](https://gemini.circumlunar.space/).

**Spec version supported:** v0.14.1, June 15th 2020

This version of the library was forked from [~yotam/go-gemini](https://git.sr.ht/~yotam/go-gemini/) to add additional features, as well as update it to support newer specs. At the time of forking, it had not seen any new commit for 5 months, and was based on v0.9.2. If there are any future upstream updates, I will make an effort to include them.

At the moment, this library focuses more on the client side of things, and support is not guaranteed. Please feel free to file issues though.

## Improvements
This fork of the library improves on the original in several ways, some listed above already.

- Client supports self-signed certs sent by the server, but still has other checks like expiry date and hostname
  - The original library could only work with self-signed certs by disabling all security.
- Invalid status code numbers raise an error
- Set default port and scheme for client requests
- Raise error when META strings are too long in the response header
- Supports new status code updates
- If `SSLKEYLOGFILE` is set, session keys are written to the file in NSS format. This is useful for debugging TLS connections (but breaks security, so don't use unless necessary).

## License
This library is under the ISC License, see the [LICENSE](./LICENSE) file for details.

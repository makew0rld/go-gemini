# go-gemini

go-gemini is a library that provides an easy interface to create client and servers that speak the [Gemini protocol](https://gemini.circumlunar.space/).

This version of the library was forked from [~yotam/go-gemini](https://git.sr.ht/~yotam/go-gemini/) to add additional features, as well as update it to support spec v0.12.3. At the time of forking, it had not seen any new commit for 5 months, and was based on v0.9.2. If there are any future upstream updates, I will make an effort to include them.

At the moment, this library focuses more on the client side of things, and support is not guaranteed. Please feel free to file issues though.

## Improvements
This fork of the library improves on the original in several ways, some listed above already.

- Client supports self-signed certs sent by the server, but still has other checks like expiry date and hostname
  - The original library could only work with self-signed certs by disabling all security.
- Invalid status code numbers raise an error
- Set default port and scheme for client requests
- Raise error when META strings are too long in the response header

## Example Server
The repository comes with an example server that respond with an hardcoded text
to the root page. To build the server run the following command:

    make build

## License
This library is under the ISC License, see the [LICENSE](./LICENSE) file for details.

# lipstick

This is a simple audio codec for VoIP applications. It mostly uses outdated techniques.

The point of this is not to compete with the best, but rather:

1) Have a relatively small implementation that is easy to understand and recreate in other programming languages.
2) Be a library without external dependencies like libopus or libvorbis and therefore make it simple to use native
Golang's cross-compilation and avoid complicating the build process in general.

Both encoder and decoder are implemented.

If you need a high-quality production-ready codec, consider using [Opus](https://github.com/hraban/opus) instead.

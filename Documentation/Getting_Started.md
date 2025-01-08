# Getting Started with GooeyLife

## Binary releases

The easiest way to install the program is to see if we have a binary on the
[Releases](https://github.com/pneumaticdeath/GooeyLife/releases) page. Unfortunately
I don't currently have a windows build environment, so I have no pre-packaged 
binaries for Microsoft Windows.  MacOS builds aren't signed because my application
to the Apple Developer program is still pending.

## Building from source

The next easiest way is to build the packages from the Go repository system.

First you need to [download and install](https://go.dev/doc/install) the latest release of the
Go environment from the official repository, or I highly recommend using [Homebrew](https://brew.sh/) on Mac or Linux
to make installing a wide variety of things easier. (After installing homebrew, you can just type `brew install go`
and let the magic happen.)

Once you've installed Go, you can test it by opening a terminal (Linux or Mac), 
or **PowerShell** or `cmd` on Windows, and typing 
```
go version
```

On my mac it looks like
```
titania:~ mitch$ go version
go version go1.23.4 darwin/arm64
```
and on my Ubuntu box is looks like 
```
mitch@phobos:~$ go version
go version go1.22.2 linux/amd64
```
(Kudos if you've figured out my machine naming scheme.)  The version I have on Linux is older only
because I'm using the pre-packaged binaries that are available for my distribution, but the code
works fine either way, as we'll see in a moment.

Now you can install GooeyLife by typing
```
go install github.com/pneumaticdeath/GooeyLife@latest
```

If your version of Go is too old, it will automatically switch it for you, like so...
```
mitch@phobos:~$ go install github.com/pneumaticdeath/GooeyLife@latest
go: downloading github.com/pneumaticdeath/GooeyLife v0.1.4
go: github.com/pneumaticdeath/GooeyLife@v0.1.4 requires go >= 1.23.4; switching to go1.23.4
```

Now you should be able to type `GooeyLife` into your command prompt to run it.  If it doesn't work, make sure
you've set up the PATH environment variable for your installation.

Now you can take a [tour](Tour.md) of the program's features.

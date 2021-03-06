## Installation

### Install Go 1.14 or higher

Follow the official docs or use your favorite dependency manager
to install Go: [https://golang.org/doc/install](https://golang.org/doc/install)

Verify your `$GOPATH` is correctly set before continuing!

### Setup this repository

Go is bit picky about where you store your repositories.

The convention is to store:

- the source code inside the `$GOPATH/src`
- the compiled program binaries inside the `$GOPATH/bin`

#### Using Git

```bash
mkdir -p $GOPATH/src/github.com/web3coach
cd $GOPATH/src/github.com/web3coach

git clone https://github.com/IacopoMelani/the-blockchain-pub.git
```

PS: Make sure you actually clone it inside the `src/github.com/web3coach` directory, not your own, otherwise it won't compile. Go rules.

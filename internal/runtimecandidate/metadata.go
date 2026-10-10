// Package runtimecandidate pins the published Vere release image and bootstrap
// pill used by the development runner. Omabit does not build or overlay Vere.
package runtimecandidate

const (
	ManifestDigest = "sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd"
	Image          = "docker.io/tloncorp/vere@" + ManifestDigest
	ConfigDigest   = "sha256:d6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245"
	Platform       = "linux/amd64"
	VereVersion    = "4.6-5a7614f"
	PillDefaultURL = "https://bootstrap.urbit.org/urbit-v4.6.pill"
	PillGeneration = "1790171686269740"
	PillURL        = PillDefaultURL + "?generation=" + PillGeneration
	PillSHA256     = "f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b"
	PillBytes      = int64(217421172)
	PillName       = "urbit-v4.6.pill"
)

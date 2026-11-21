// Package runtimecandidate pins the published Vere release image and bootstrap
// pill used by the development runner. Omabit does not build or overlay Vere.
package runtimecandidate

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"

const (
	ManifestDigest = runtimeprofile.ManifestDigest
	Image          = runtimeprofile.Image
	ConfigDigest   = runtimeprofile.ConfigDigest
	Platform       = runtimeprofile.Platform
	VereVersion    = runtimeprofile.VereVersion
	PillDefaultURL = "https://bootstrap.urbit.org/urbit-v4.6.pill"
	PillGeneration = "1790171686269740"
	PillURL        = PillDefaultURL + "?generation=" + PillGeneration
	PillSHA256     = "f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b"
	PillBytes      = int64(217421172)
	PillName       = "urbit-v4.6.pill"
)

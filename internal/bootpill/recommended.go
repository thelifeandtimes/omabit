package bootpill

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"

// These release pins are trusted inputs from the accepted checkout/release
// channel, not hashes learned from a download. The provider URL remains a known
// alias so existing desktop preferences select the same frozen release artifact.
const RecommendedAlias = "https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill"
const RecommendedURL = "https://github.com/thelifeandtimes/omabit/releases/download/v0.1.0-rc.11/omabit-recommended-apps-0.1.0-rc.11.pill"

func RecommendedDescriptor() protocol.BootPill {
	return protocol.BootPill{SHA256: "85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62", Bytes: 59351751}
}

func IsRecommended(source string) bool {
	return source == RecommendedAlias || source == RecommendedURL
}

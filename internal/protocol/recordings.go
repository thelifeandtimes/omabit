package protocol

import "time"

const RecordingReadFeature = "runtime-recording-read-v1"
const RecordingCatalogPath = "/v1/recordings/catalog"
const RecordingPagePath = "/v1/recordings/page"

const RecordingExportFeature = "runtime-recording-export-v1"
const RecordingExportManifestPath = "/v1/recordings/export/manifest"
const RecordingExportPartPath = "/v1/recordings/export/part"
const RecordingExportTimeout = 60 * time.Second

const RecordingControlFeature = "runtime-recording-control-v1"
const RecordingControlStatusPath = "/v1/recordings/controls/status"
const RecordingControlApplyPath = "/v1/recordings/controls/apply"
const RecordingControlLookupPath = "/v1/recordings/controls/lookup"

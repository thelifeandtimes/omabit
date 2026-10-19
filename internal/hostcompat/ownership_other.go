//go:build !linux

package hostcompat

import "io/fs"

func trustedInfo(info fs.FileInfo) bool { return false }

func sameFileState(a, b fs.FileInfo) bool { return false }

func stampFor(info fs.FileInfo) (fileStamp, bool) { return fileStamp{}, false }

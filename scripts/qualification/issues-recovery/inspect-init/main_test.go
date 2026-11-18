//go:build linux && amd64

package main

import "testing"

func TestInspectionRequiresExactKernelTokenIncludingLastToken(t *testing.T) {
	const marker = "omabit.disposable-guest=issues-readonly-inspection-v1"
	for _, command := range []string{marker, "console=ttyS0 " + marker + "\n", marker + " quiet"} {
		if !inspectionGuest(command, "QEMU\n") {
			t.Fatal("exact marker refused", command)
		}
	}
	for _, command := range []string{"", "prefix-" + marker, marker + "-suffix"} {
		if inspectionGuest(command, "QEMU") {
			t.Fatal("nonmatching marker admitted")
		}
	}
	if inspectionGuest(marker, "physical-host") {
		t.Fatal("non-QEMU host admitted")
	}
}

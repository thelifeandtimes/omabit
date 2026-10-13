package pierstore

import "testing"

// A reboot can renumber btrfs anonymous devices (and dm/loop block devices);
// only a different inode or birth timestamp means a different directory.
func TestSameObjectIgnoresVolatileDeviceNumbers(t *testing.T) {
	stored := Identity{Device: 30, Inode: 4100, UrbDevice: 30, UrbInode: 4101, Birth: "1791760291:850000001", UrbBirth: "1791760292:12"}
	rebooted := stored
	rebooted.Device, rebooted.UrbDevice = 31, 31
	if !stored.SameObject(rebooted) || !rebooted.SameObject(stored) {
		t.Fatal("device renumbering was treated as a replaced pier")
	}
	for name, mutate := range map[string]func(*Identity){
		"inode":     func(i *Identity) { i.Inode++ },
		"birth":     func(i *Identity) { i.Birth = "1791760291:850000002" },
		"urb inode": func(i *Identity) { i.UrbInode++ },
		"urb birth": func(i *Identity) { i.UrbBirth = "1791760292:13" },
	} {
		replaced := rebooted
		mutate(&replaced)
		if stored.SameObject(replaced) {
			t.Fatal("replaced pier accepted after", name, "change")
		}
	}

	mount := DirectoryIdentity{Device: 30, Inode: 4099, Birth: "1791760291:1"}
	if !mount.SameObject(DirectoryIdentity{Device: 58, Inode: 4099, Birth: "1791760291:1"}) {
		t.Fatal("mount root device renumbering was treated as replacement")
	}
	if mount.SameObject(DirectoryIdentity{Device: 30, Inode: 4099, Birth: "1791760291:2"}) || mount.SameObject(DirectoryIdentity{Device: 30, Inode: 4098, Birth: "1791760291:1"}) {
		t.Fatal("replaced mount root accepted")
	}
}

// Records without birth timestamps cannot prove object identity without st_dev.
func TestSameObjectKeepsStrictComparisonWithoutBirth(t *testing.T) {
	legacy := Identity{Device: 1, Inode: 2, UrbDevice: 1, UrbInode: 3}
	moved := legacy
	moved.Device = 9
	if !legacy.SameObject(legacy) || legacy.SameObject(moved) {
		t.Fatal("legacy identity comparison changed")
	}
	if (DirectoryIdentity{Device: 1, Inode: 4}).SameObject(DirectoryIdentity{Device: 2, Inode: 4}) {
		t.Fatal("legacy mount identity comparison changed")
	}
}

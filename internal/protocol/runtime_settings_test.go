package protocol

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

func TestNativeControlWarningMatchesUI(t *testing.T) {
	data, err := os.ReadFile("../../plugins/omabit/RuntimeSettingsFields.qml")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?s)objectName:\s*"runtime-native-warning".*?text:\s*("(?:\\.|[^"\\])*")`).FindSubmatch(data)
	if len(match) != 2 {
		t.Fatal("native-control UI warning is not a literal text field")
	}
	text, err := strconv.Unquote(string(match[1]))
	if err != nil {
		t.Fatal(err)
	}
	if text != NativeControlWarning {
		t.Fatalf("CLI and UI native-control warnings differ:\nCLI: %q\nUI: %q", NativeControlWarning, text)
	}
}

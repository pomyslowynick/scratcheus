package labels

import "testing"

func Test_label_bytes(t *testing.T) {
	label := Label{
		Name:  "status_code",
		Value: "500",
	}

	labelAsBytes := label.Bytes()

	if string(labelAsBytes) != "status_code500" {
		t.Errorf("Returned bytes are not equal to the label name/value")
	}
}

package messages

import (
	"testing"
	"time"
)

func TestClampTTL(t *testing.T) {
	if ClampTTL(-1) != 0 { t.Fatal("negative should be 0") }
	if ClampTTL(0) != 0 { t.Fatal("zero should be 0") }
	if ClampTTL(time.Hour) != time.Hour { t.Fatal("1h should pass") }
	if ClampTTL(40*24*time.Hour) != MaxTTL { t.Fatal(">30d should clamp") }
}
package economysim_test

import (
	"os"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// TestConfig_FileAndGoAgree verifies that the pack one loaded from its three
// layers in testdata/chains is, field for field, the reference configuration
// built in Go, and that the two give the same hash: a pack is data, and the
// file layout is only a way to write the same configuration.
//
// Requirement: FR-CFG-004, FR-CFG-025
func TestConfig_FileAndGoAgree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	fromFile, err := config.LoadFS(os.DirFS("testdata/chains"), "base.toml", "one.toml", "warm-one.toml")
	if err != nil {
		t.Fatalf("LoadFS(base, one, warm-one) error = %v, want nil", err)
	}
	fromGo := referenceConfig()
	file, goV := reflect.ValueOf(fromFile), reflect.ValueOf(fromGo)
	for i := range file.NumField() {
		name := file.Type().Field(i).Name
		if !reflect.DeepEqual(file.Field(i).Interface(), goV.Field(i).Interface()) {
			t.Errorf("Config.%s from the file = %+v, want the Go reference %+v", name, file.Field(i).Interface(), goV.Field(i).Interface())
		}
	}
	if got, want := economysim.HashConfig(fromFile), economysim.HashConfig(fromGo); got != want {
		t.Errorf("HashConfig(file) = %x, want %x (HashConfig of the Go reference)", got, want)
	}
}

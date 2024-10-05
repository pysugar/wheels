package features

import (
	"github.com/pysugar/wheels/lang"
	"log"
)

// Feature is the interface for Xray features. All features must implement this interface.
// All existing features have an implementation in app directory. These features can be replaced by third-party ones.
type Feature interface {
	lang.HasType
	lang.Runnable
}

// PrintDeprecatedFeatureWarning prints a warning for deprecated feature.
func PrintDeprecatedFeatureWarning(feature string) {
	log.Println("You are using a deprecated feature: " + feature + ". Please update your config file with latest configuration format, or update your client software.")
}

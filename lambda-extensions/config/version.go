package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExtensionName same as binary name or file name where main exists
var ExtensionName = filepath.Base(os.Args[0])
var layerVersion = "2"

// SumoLogicExtensionLayerVersionSuffix denotes the layer version published in AWS
var SumoLogicExtensionLayerVersionSuffix string = fmt.Sprintf("%s-prod:%s", ExtensionName, layerVersion)

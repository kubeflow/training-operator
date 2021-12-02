package config

// Config is the global configuration for the training operator.
var Config struct {
	PyTorchInitContainerTemplateFile string
	PyTorchInitContainerImage        string
}

const (
	// PyTorchInitContainerImageDefault is the default image for the pytorch
	// init container.
	PyTorchInitContainerImageDefault = "alpine:3.10"
	// PyTorchInitContainerTemplateFileDefault is the default template file for
	// the pytorch init container.
	PyTorchInitContainerTemplateFileDefault = "/etc/config/initContainer.yaml"
)

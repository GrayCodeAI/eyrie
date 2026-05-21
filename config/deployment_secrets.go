package config

// SanitizeDeploymentConfigForDisk removes secret fields before writing provider.json.
// Runtime clients resolve API keys from the process environment or credential store.
func SanitizeDeploymentConfigForDisk(dc DeploymentConfig) DeploymentConfig {
	dc.APIKey = ""
	dc.Token = ""
	dc.SecretAccessKey = ""
	dc.AccessKeyID = ""
	dc.SessionToken = ""
	return dc
}

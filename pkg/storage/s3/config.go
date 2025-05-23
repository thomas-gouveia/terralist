package s3

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	prefixRegEx = regexp.MustCompile(`(?m)^[a-zA-Z0-9\(\)\'\*\.\-_\!\/]+$`)
)

// Config implements storage.Configurator interface and
// handles the configuration parameters of the s3 resolver.
type Config struct {
	Endpoint string

	BucketName      string
	BucketRegion    string
	BucketPrefix    string
	AccessKeyID     string
	SecretAccessKey string

	ServerSideEncryption string
	UsePathStyle         bool

	LinkExpire         int
	DefaultCredentials bool
}

func (c *Config) SetDefaults() {}

func (c *Config) Validate() error {
	if c.BucketName == "" {
		return fmt.Errorf("missing required attribute 'BucketName'")
	}

	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		c.DefaultCredentials = true
	} else {
		c.DefaultCredentials = false
	}

	if c.BucketPrefix != "" {
		if strings.HasPrefix(c.BucketPrefix, "/") {
			return fmt.Errorf("the prefix must not start with a slash ('/')")
		}

		if strings.HasSuffix(c.BucketPrefix, "/") {
			return fmt.Errorf("the prefix must not end with a slash ('/')")
		}

		if !prefixRegEx.MatchString(c.BucketPrefix) {
			return fmt.Errorf("the prefix contains invalid characters")
		}

		c.BucketPrefix = fmt.Sprintf("%s/", c.BucketPrefix)
	}

	if c.LinkExpire <= 0 {
		return fmt.Errorf("the expire time for links must be positive > 0")
	}

	return nil
}

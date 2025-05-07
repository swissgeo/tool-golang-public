package aws_test

import (
	"testing"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Only minimal test
func TestGetLocalAwsProfiles(t *testing.T) {
	got, err := aws.GetLocalBgdiAdminProfiles()
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

package aws_test

import (
	"testing"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Only minimal test
func TestGetLocalAdminProfiles(t *testing.T) {
	got, err := aws.GetLocalAdminProfiles()
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

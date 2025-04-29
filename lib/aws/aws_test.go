package aws_test

import (
	"fmt"
	"testing"

	"github.com/geoadmin/tool-golang-bgdi/lib/aws"
	"github.com/stretchr/testify/require"
)

func TestGetLocalAwsProfiles(t *testing.T) {
	// TODO: Write propper tests
	got, err := aws.GetLocalBgdiAdminProfiles()
	require.NoError(t, err)
	fmt.Println(got)
}

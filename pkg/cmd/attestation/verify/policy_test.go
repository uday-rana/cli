package verify

import (
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact/oci"
	"github.com/cli/cli/v2/pkg/cmd/factory"

	"github.com/stretchr/testify/require"
)

func TestNewEnforcementCriteria(t *testing.T) {
	artifactPath := "../test/data/sigstore-js-2.1.0.tgz"

	artifact, err := artifact.NewDigestedArtifact(oci.MockClient{}, artifactPath, "sha256")
	require.NoError(t, err)

	t.Run("sets SANRegex using SignerRepo", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
			SignerRepo:   "foo/bar",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "^https://github.com/foo/bar", c.Extensions.SANRegex)
		require.Zero(t, c.Extensions.SAN)
	})

	t.Run("sets SANRegex using SignerWorkflow", func(t *testing.T) {
		opts := &Options{
			ArtifactPath:   artifactPath,
			Owner:          "foo",
			Repo:           "bar",
			SignerWorkflow: "foo/bar/.github/workflows/attest.yml",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "^https://github.com/foo/bar/.github/workflows/attest.yml", c.Extensions.SANRegex)
		require.Zero(t, c.Extensions.SAN)
	})

	t.Run("sets SANRegex and SAN using SANRegex and SAN", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
			SAN:          "https://github/foo/bar/.github/workflows/attest.yml",
			SANRegex:     "^https://github/foo",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://github/foo/bar/.github/workflows/attest.yml", c.Extensions.SANRegex)
		require.Equal(t, "^https://github/foo", c.Extensions.SAN)
	})

	t.Run("sets Extensions.RunnerEnvironment to GitHubRunner value if opts.DenySelfHostedRunner is true", func(t *testing.T) {
		opts := &Options{
			ArtifactPath:         artifactPath,
			Owner:                "foo",
			Repo:                 "bar",
			DenySelfHostedRunner: true,
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, GitHubRunner, c.Extensions.RunnerEnvironment)
	})

	t.Run("sets Extensions.RunnerEnvironment to * value if opts.DenySelfHostedRunner is false", func(t *testing.T) {
		opts := &Options{
			ArtifactPath:         artifactPath,
			Owner:                "foo",
			Repo:                 "bar",
			DenySelfHostedRunner: false,
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "*", c.Extensions.RunnerEnvironment)
	})

	t.Run("sets Extensions.BuildSourceRepoURI using opts.Repo and opts.Tenant", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
			Tenant:       "baz",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://baz.ghe.com/foo/bar", c.Extensions.BuildSourceRepoURI)
	})

	t.Run("sets Extensions.BuildSourceRepoURI using opts.Repo", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://github.com/foo/bar", c.Extensions.BuildSourceRepoURI)
	})

	t.Run("sets Extensions.SourceRepositoryOwnerURI using opts.Owner and opts.Tenant", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
			Tenant:       "baz",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://baz.ghe.com/foo", c.Extensions.SourceRepositoryOwnerURI)
	})

	t.Run("sets Extensions.SourceRepositoryOwnerURI using opts.Owner", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://github.com/foo", c.Extensions.SourceRepositoryOwnerURI)
	})

	t.Run("sets OIDCIssuer using opts.OIDCIssuer and opts.Tenant", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
			Tenant:       "baz",
			OIDCIssuer:   "https://foo.com",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://token.actions.baz.ghe.com", c.OIDCIssuer)
	})

	t.Run("sets OIDCIssuer using opts.OIDCIssuer", func(t *testing.T) {
		opts := &Options{
			ArtifactPath: artifactPath,
			Owner:        "foo",
			Repo:         "bar",
			OIDCIssuer:   "https://foo.com",
		}

		c, err := newEnforcementCriteria(opts, *artifact)
		require.NoError(t, err)
		require.Equal(t, "https://foo.com", c.OIDCIssuer)
	})
}

func TestValidateSignerWorkflow(t *testing.T) {
	type testcase struct {
		name                   string
		providedSignerWorkflow string
		expectedWorkflowRegex  string
		host                   string
		expectErr              bool
		errContains            string
	}

	testcases := []testcase{
		{
			name:                   "workflow with no host specified",
			providedSignerWorkflow: "github/artifact-attestations-workflows/.github/workflows/attest.yml",
			expectErr:              true,
			errContains:            "unknown host",
		},
		{
			name:                   "workflow with default host",
			providedSignerWorkflow: "github/artifact-attestations-workflows/.github/workflows/attest.yml",
			expectedWorkflowRegex:  "^https://github.com/github/artifact-attestations-workflows/.github/workflows/attest.yml",
			host:                   "github.com",
		},
		{
			name:                   "workflow with workflow URL included",
			providedSignerWorkflow: "github.com/github/artifact-attestations-workflows/.github/workflows/attest.yml",
			expectedWorkflowRegex:  "^https://github.com/github/artifact-attestations-workflows/.github/workflows/attest.yml",
			host:                   "github.com",
		},
		{
			name:                   "workflow with GH_HOST set",
			providedSignerWorkflow: "github/artifact-attestations-workflows/.github/workflows/attest.yml",
			expectedWorkflowRegex:  "^https://myhost.github.com/github/artifact-attestations-workflows/.github/workflows/attest.yml",
			host:                   "myhost.github.com",
		},
		{
			name:                   "workflow with authenticated host",
			providedSignerWorkflow: "github/artifact-attestations-workflows/.github/workflows/attest.yml",
			expectedWorkflowRegex:  "^https://authedhost.github.com/github/artifact-attestations-workflows/.github/workflows/attest.yml",
			host:                   "authedhost.github.com",
		},
	}

	for _, tc := range testcases {
		opts := &Options{
			Config:         factory.New("test").Config,
			SignerWorkflow: tc.providedSignerWorkflow,
		}

		// All host resolution is done verify.go:RunE
		opts.Hostname = tc.host
		workflowRegex, err := validateSignerWorkflow(opts)
		require.Equal(t, tc.expectedWorkflowRegex, workflowRegex)

		if tc.expectErr {
			require.Error(t, err)
			require.ErrorContains(t, err, tc.errContains)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.expectedWorkflowRegex, workflowRegex)
		}
	}
}

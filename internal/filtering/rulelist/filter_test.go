package rulelist_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdGuardPrivate/AdGuardPrivate/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilter_Refresh(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()

	const fltData = testRuleTextTitle + testRuleTextBlocked
	fileURL, srvURL := newFilterLocations(t, cacheDir, fltData, fltData)

	testCases := []struct {
		url           *url.URL
		name          string
		wantNewErrMsg string
	}{{
		url:           nil,
		name:          "nil_url",
		wantNewErrMsg: "no url",
	}, {
		url: &url.URL{
			Scheme: "ftp",
		},
		name:          "bad_scheme",
		wantNewErrMsg: `bad url scheme: "ftp"`,
	}, {
		name: "file",
		url: &url.URL{
			Scheme: urlutil.SchemeFile,
			Path:   fileURL.Path,
		},
		wantNewErrMsg: "",
	}, {
		name:          "http",
		url:           srvURL,
		wantNewErrMsg: "",
	}}

	for _, tc := range testCases {
		tc := tc // 捕获循环变量
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 每个子测试使用独立的 UID
			uid := rulelist.MustNewUID()

			f, err := rulelist.NewFilter(&rulelist.FilterConfig{
				URL:         tc.url,
				Name:        tc.name,
				UID:         uid,
				URLFilterID: testURLFilterID,
				Enabled:     true,
			})
			if tc.wantNewErrMsg != "" {
				assert.EqualError(t, err, tc.wantNewErrMsg)

				return
			}

			testutil.CleanupAndRequireSuccess(t, f.Close)

			require.NotNil(t, f)

			buf := make([]byte, rulelist.DefaultRuleBufSize)
			cli := &http.Client{
				Timeout: testTimeout,
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			res, err := f.Refresh(ctx, buf, cli, cacheDir, rulelist.DefaultMaxRuleListSize)
			require.NoError(t, err)

			assert.Equal(t, testTitle, res.Title)
			assert.Equal(t, len(testRuleTextBlocked), res.BytesWritten)
			assert.Equal(t, 1, res.RulesCount)

			// Check that the cached file exists.
			_, err = os.Stat(filepath.Join(cacheDir, uid.String()+".txt"))
			require.NoError(t, err)
		})
	}
}

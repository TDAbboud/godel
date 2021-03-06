// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package godelgetter_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/mholt/archiver/v3"
	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/godel/v2/godelgetter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadIntoDirectory(t *testing.T) {
	for i, tc := range []struct {
		setup func(t *testing.T, repoTGZFile string) (srcPath string, cleanup func())
	}{
		{
			func(t *testing.T, repoTGZFile string) (srcPath string, cleanup func()) {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					bytes, err := os.ReadFile(repoTGZFile)
					require.NoError(t, err)
					_, err = w.Write(bytes)
					require.NoError(t, err)
				}))
				return ts.URL + "/test-on-server.tgz", ts.Close
			},
		},
		{
			func(t *testing.T, repoTGZFile string) (srcPath string, cleanup func()) {
				return repoTGZFile, nil
			},
		},
	} {
		func() {
			tmpDir, cleanup, err := dirs.TempDir("", "")
			defer cleanup()
			require.NoError(t, err)

			repoDir := filepath.Join(tmpDir, "repo")
			err = os.MkdirAll(repoDir, 0755)
			require.NoError(t, err, "Case %d", i)

			downloadsDir := filepath.Join(tmpDir, "downloads")
			err = os.MkdirAll(downloadsDir, 0755)
			require.NoError(t, err, "Case %d", i)

			repoTGZFile := filepath.Join(repoDir, "test.tgz")
			writeSimpleTestTgz(t, repoTGZFile)

			srcPath, cleanup := tc.setup(t, repoTGZFile)
			if cleanup != nil {
				defer cleanup()
			}

			outBytes := &bytes.Buffer{}
			fileName, err := godelgetter.DownloadIntoDirectory(godelgetter.NewPkgSrc(srcPath, ""), downloadsDir, outBytes)
			require.NoError(t, err, "Case %d", i)

			err = archiver.DefaultTarGz.Unarchive(fileName, tmpDir)
			require.NoError(t, err, "Case %d", i)

			fileBytes, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
			require.NoError(t, err, "Case %d", i)

			assert.Equal(t, "Test file\n", string(fileBytes), "Case %d", i)
			assert.Regexp(t, fmt.Sprintf("(?s)Getting package from %s", srcPath)+regexp.QuoteMeta("...")+".+", outBytes.String(), "Case %d", i)
		}()
	}
}

func TestDownloadSameFileOK(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.txt")
	const content = "test content"
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	_, err = godelgetter.DownloadIntoDirectory(godelgetter.NewPkgSrc(testFile, ""), tmpDir, io.Discard)
	require.NoError(t, err)

	gotContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, content, string(gotContent))
}

func TestFailedReaderDoesNotOverwriteDestinationFile(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	const fileName = "test.txt"

	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	dstDir := filepath.Join(tmpDir, "dst")
	err = os.MkdirAll(dstDir, 0755)
	require.NoError(t, err)

	// write content in destination file
	dstFile := filepath.Join(dstDir, fileName)
	const content = "destination content"
	err = os.WriteFile(dstFile, []byte(content), 0644)
	require.NoError(t, err)

	srcFile := filepath.Join(srcDir, fileName)
	_, err = godelgetter.DownloadIntoDirectory(godelgetter.NewPkgSrc(srcFile, ""), dstDir, os.Stdout)
	// download operation should fail
	assert.EqualError(t, err, fmt.Sprintf("%s does not exist", srcFile))

	// destination file should still exist (should not have been truncated by download with failed source reader)
	gotContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(gotContent))
}

func writeSimpleTestTgz(t *testing.T, filePath string) {
	tmpDir, cleanup, err := dirs.TempDir("", "")
	defer cleanup()
	require.NoError(t, err)

	testFilePath := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFilePath, []byte("Test file\n"), 0644)
	require.NoError(t, err)

	err = archiver.DefaultTarGz.Archive([]string{testFilePath}, filePath)
	require.NoError(t, err)
}

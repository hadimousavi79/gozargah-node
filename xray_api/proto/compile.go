package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Helper function to check for errors
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Function to download the latest release tarball
func downloadLatestRelease(url string, dest string) {
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("Failed to download file: %s", resp.Status))
	}

	out, err := os.Create(dest)
	check(err)
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	check(err)
}

// Function to extract tarball
func extractTarGz(src, dest string) {
	file, err := os.Open(src)
	check(err)
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	check(err)
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		check(err)

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0755)
			check(err)
		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(target), 0755)
			check(err)
			outFile, err := os.Create(target)
			check(err)
			defer outFile.Close()
			_, err = io.Copy(outFile, tr)
			check(err)
		}
	}
}

// Function to copy .pb.go files to the destination directory and modify imports
func copyAndModifyPbGoFiles(srcDir, destDir, oldPrefix, newPrefix string) {
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		check(err)
		if strings.HasSuffix(info.Name(), ".pb.go") {
			relPath, err := filepath.Rel(srcDir, path)
			check(err)
			destPath := filepath.Join(destDir, relPath)
			err = os.MkdirAll(filepath.Dir(destPath), 0755)
			check(err)
			_, err = copyAndModifyFile(path, destPath, oldPrefix, newPrefix)
			check(err)
		}
		return nil
	})
	check(err)
}

// Function to copy a file and modify imports
func copyAndModifyFile(src, dest, oldPrefix, newPrefix string) (int64, error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer destFile.Close()

	content, err := io.ReadAll(sourceFile)
	if err != nil {
		return 0, err
	}

	modifiedContent := modifyImports(string(content), oldPrefix, newPrefix)
	_, err = destFile.Write([]byte(modifiedContent))
	if err != nil {
		return 0, err
	}

	return int64(len(modifiedContent)), nil
}

// Function to modify import paths
func modifyImports(content, oldPrefix, newPrefix string) string {
	re := regexp.MustCompile(`(` + regexp.QuoteMeta(oldPrefix) + `)`)
	return re.ReplaceAllString(content, newPrefix)
}

func main() {
	// Get latest version tag from GitHub API
	resp, err := http.Get("https://api.github.com/repos/XTLS/xray-core/releases/latest")
	check(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("Failed to fetch latest release: %s", resp.Status))
	}

	type GitHubRelease struct {
		TagName string `json:"tag_name"`
	}

	var release GitHubRelease
	err = json.NewDecoder(resp.Body).Decode(&release)
	check(err)

	version := release.TagName
	fmt.Println("Latest version is", version)

	// Download source tarball
	downloadUrl := fmt.Sprintf("https://github.com/XTLS/xray-core/archive/refs/tags/%s.tar.gz", version)
	tmpDir, err := os.MkdirTemp("", "xray-core-*")
	check(err)
	defer os.RemoveAll(tmpDir)

	tarballPath := filepath.Join(tmpDir, "source.tar.gz")
	fmt.Println("Downloading source", version, "...")
	downloadLatestRelease(downloadUrl, tarballPath)
	fmt.Println("Source downloaded. Extracting...")

	// Extract tarball
	extractDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(extractDir, 0755)
	check(err)
	extractTarGz(tarballPath, extractDir)
	fmt.Println("Source extracted.")

	// Copy .pb.go files to the destination directory and modify imports
	destDir := "." // or any directory you want to copy files to
	oldPrefix := "github.com/xtls/xray-core"
	newPrefix := "marzban-node/xray_api/proto"

	fmt.Println("Copying .pb.go files and modifying imports...")
	copyAndModifyPbGoFiles(extractDir, destDir, oldPrefix, newPrefix)
	fmt.Println("Done.")
}

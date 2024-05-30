package loader

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/pkg/errors"
)

var drivePathPattern = regexp.MustCompile(`^[a-zA-Z]:/`)

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

// BufferedFile represents an archive file buffered for later processing.
type BufferedFile struct {
	Name string
	Data []byte
}

func ArchiveSource(text string, name string) *pkl.ModuleSource {
	return &pkl.ModuleSource{
		// repl:text
		Uri: &url.URL{
			Scheme: "oci",
			Path:   name,
		},
		Contents: text,
	}
}

// LoadArchiveFiles reads in files out of an archive into memory. This function
// performs important path security checks and should always be used before
// expanding a tarball
func LoadArchiveFiles(in io.Reader) ([]*BufferedFile, error) {
	unzipped, err := gzip.NewReader(in)
	if err != nil {
		return nil, err
	}
	defer unzipped.Close()

	files := []*BufferedFile{}
	tr := tar.NewReader(unzipped)
	for {
		b := bytes.NewBuffer(nil)
		hd, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hd.FileInfo().IsDir() {
			// Use this instead of hd.Typeflag because we don't have to do any
			// inference chasing.
			continue
		}

		switch hd.Typeflag {
		// We don't want to process these extension header files.
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		}

		// Archive could contain \ if generated on Windows
		delimiter := "/"
		if strings.ContainsRune(hd.Name, '\\') {
			delimiter = "\\"
		}

		parts := strings.Split(hd.Name, delimiter)
		n := strings.Join(parts[1:], delimiter)

		// Normalize the path to the / delimiter
		n = strings.ReplaceAll(n, delimiter, "/")

		if path.IsAbs(n) {
			return nil, errors.New("package illegally contains absolute paths")
		}

		n = path.Clean(n)
		if n == "." {
			// In this case, the original path was relative when it should have been absolute.
			return nil, errors.Errorf("package illegally contains content outside the base directory: %q", hd.Name)
		}
		if strings.HasPrefix(n, "..") {
			return nil, errors.New("package illegally references parent directory")
		}

		// In some particularly arcane acts of path creativity, it is possible to intermix
		// UNIX and Windows style paths in such a way that you produce a result of the form
		// c:/foo even after all the built-in absolute path checks. So we explicitly check
		// for this condition.
		if drivePathPattern.MatchString(n) {
			return nil, errors.New("package contains illegally named files")
		}

		if parts[0] == "hpkl.pkl" {
			return nil, errors.New("package metadata not in base directory")
		}

		if _, err := io.Copy(b, tr); err != nil {
			return nil, err
		}

		data := bytes.TrimPrefix(b.Bytes(), utf8bom)

		files = append(files, &BufferedFile{Name: n, Data: data})
		b.Reset()
	}

	if len(files) == 0 {
		return nil, errors.New("no files in package archive")
	}
	return files, nil
}

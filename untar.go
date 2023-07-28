package main

import (
	"archive/tar"
	"compress/bzip2"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
)

var exts = map[string]string{
	".tar":  "tar",
	".gz":   "gzip",
	".tgz":  "gzip",
	".taz":  "gzip",
	".bz2":  "bzip2",
	".tz2":  "bzip2",
	".tbz2": "bzip2",
	".tbz":  "bzip2",
	".xz":   "xz",
	".zst":  "zstd",
	".tzst": "zstd",
	".lzma": "lzma",
	".tlz":  "lzma",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: untar <filename>")
		return
	}

	filename := os.Args[1]
	ext := filepath.Ext(filename)
	compress, ok := exts[ext]
	if !ok {
		fmt.Printf("invalid extension %s\n", ext)
		return
	} else {
		fmt.Printf("[INFO] Detected \"%s\" compression.\n", compress)
	}

	var reader io.Reader

	dst, _ := os.Getwd()
	fname, _, _ := strings.Cut(filename, ".")
	dst = filepath.Join(dst, fname)

	buf, err := os.Open(filename)
	if err != nil {
		fsErr := err.(*fs.PathError)
		if fsErr.Err == syscall.ENOENT {
			fmt.Printf("the file %s does not exist.", fsErr.Path)
		}
		return
	}

	switch compress {
	case "gzip":
		var err error
		reader, err = pgzip.NewReader(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
	case "bzip2":
		reader = bzip2.NewReader(buf)
	case "xz":
		var err error
		reader, err = xz.NewReader(buf)
		if err != nil {
			panic(err)
		}
	case "lzma":
		var err error
		reader, err = lzma.NewReader(buf)
		if err != nil {
			panic(err)
		}
	case "zstd":
		var err error
		reader, err = zstd.NewReader(buf)
		if err != nil {
			panic(err)
		}
	case "tar":
		reader = buf
	default:
		fmt.Println("invalid extension.")
	}

	tarReader := tar.NewReader(reader)

	for {
		if tarReader == nil {
			fmt.Println("tarReader is nil.")
			break
		}
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if header == nil {
			continue
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {

		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					panic(err)
				}
				fmt.Printf("[INFO] Created folder -> %s.\n", target)
			}

		case tar.TypeReg:
			_, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				if err := os.MkdirAll(filepath.Dir(target), 0770); err != nil {
					panic(err)
				}
				fmt.Printf("[INFO] Created folder -> %s.\n", filepath.Dir(target))
			}

			f, _ := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))

			if _, err := io.Copy(f, tarReader); err != nil {
				panic(err)
			}
			fmt.Printf("[INFO] Created file -> %s.\n", target)
			f.Close()
		}
	}
}

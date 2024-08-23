package main

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
)

var exts = map[string]string{
	".tar":  "tar",
	".zip":  "zip",
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

var filename, outdir string

func init() {
	flag.StringVar(&filename, "file", "", "To file to un-archive")
	dst := mustv(os.Getwd())
	flag.StringVar(&outdir, "outdir", dst, "The output directory")
}

func mustv[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	os.MkdirAll(outdir, 0755)

	ext := filepath.Ext(filename)
	compress, ok := exts[ext]
	if !ok {
		fmt.Printf("[ERR] Invalid extension %s.\n", ext)
		return
	} else {
		fmt.Printf("[INF] Detected \"%s\" compression.\n", compress)
	}

	var reader io.Reader

	buf := mustv(os.Open(filename))

	defer buf.Close()

	if compress == "zip" {
		extractZip(buf, outdir)
	} else {
		switch compress {
		case "gzip":
			reader = mustv(pgzip.NewReader(buf))
		case "bzip2":
			reader = bzip2.NewReader(buf)
		case "xz":
			reader = mustv(xz.NewReader(buf))
		case "lzma":
			reader = mustv(lzma.NewReader(buf))
		case "zstd":
			reader = mustv(zstd.NewReader(buf))
		case "tar":
			reader = buf
		}

		extractTar(reader, outdir)
	}
}

func extractZip(reader *os.File, dst string) {
	stat, _ := reader.Stat()
	r := mustv(zip.NewReader(reader, stat.Size()))

	for _, header := range r.File {
		if header.Mode().IsDir() {
			target := filepath.Join(dst, header.Name)

			must(os.MkdirAll(target, 0755))

			fmt.Printf("[INFO] Created folder \"%s\".\n", target)
		} else {
			target := filepath.Join(dst, header.Name)

			f := mustv(os.OpenFile(target, os.O_CREATE|os.O_RDWR, header.Mode()))

			headerBuf, _ := header.OpenRaw()

			mustv(io.Copy(f, headerBuf))

			fmt.Printf("[INFO] Created file \"%s\".\n", target)

			f.Close()
		}
	}
}

func extractTar(reader io.Reader, dst string) {
	tarReader := tar.NewReader(reader)

	for {
		if tarReader == nil {
			fmt.Println("[ERR] Could not read file.")
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
				must(os.MkdirAll(target, 0755))

				fmt.Printf("[INFO] Created folder \"%s\".\n", target)
			}

		case tar.TypeReg:
			f := mustv(os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode)))

			mustv(io.Copy(f, tarReader))

			fmt.Printf("[INFO] Created file \"%s\".\n", target)
			f.Close()
		}
	}
}

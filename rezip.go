package main

import "archive/zip"
import "bytes"
import "flag"
import "io"
import "io/ioutil"
import "log"
import "os"
import "fmt"

func main() {
    var inPath string
    var outPath string

    flag.StringVar(&inPath, "In", "", "")
    flag.StringVar(&outPath, "Out", "", "")
    flag.Parse()

    inContent, err := ioutil.ReadFile(inPath)
    if err != nil { log.Fatal(err) }

    sourceZipFile, err := zip.NewReader(bytes.NewReader(inContent), int64(len(inContent)))
    if err != nil {
        log.Fatal(err)
    }

    destinationFile, err := os.Create(outPath)
    if err != nil {
        log.Fatal(err)
    }
    defer destinationFile.Close()
    destinationZipFile := NewAPPXWriter(destinationFile)
    defer destinationZipFile.Close()

    for _, zipEntry := range sourceZipFile.File {
        zipEntryFile, err := zipEntry.Open()
        if err != nil {
            log.Fatal(err)
        }
        defer zipEntryFile.Close()

        // @@@
        zipEntry.FileHeader.Extra = nil
        zipEntry.CompressedSize = uint32(zipEntry.CompressedSize64)
        zipEntry.UncompressedSize = uint32(zipEntry.UncompressedSize64)
        fmt.Printf("%#v\n", zipEntry.FileHeader)

        rawZIPEntryFile, err := zipEntry.OpenRaw()
        if err != nil {
            log.Fatal(err)
        }
        destinationZipEntryFile, err := destinationZipFile.CreateRaw(&zipEntry.FileHeader)
        if err != nil {
            log.Fatal(err)
        }
        _, err = io.Copy(destinationZipEntryFile, rawZIPEntryFile)
        if err != nil {
            log.Fatal(err)
        }
    }
}

type APPXWriter struct {
    file io.WriterAt
}

func NewAPPXWriter(file io.WriterAt) APPXWriter {
    return APPXWriter{
        file: file,
    }
}

func (w *APPXWriter) Close() error {
    return nil
}

func (w *APPXWriter) CreateRaw(header *zip.FileHeader) (io.Writer, error) {
    return APPXRawFileWriter{
        w: w,
    }, nil
}

type APPXRawFileWriter struct {
    w *APPXWriter
}

func (w APPXRawFileWriter) Write(data []byte) (int, error) {
    return len(data), nil
}

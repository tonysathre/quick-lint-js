package main

import "archive/zip"
import "bytes"
import "encoding/binary"
import "flag"
import "fmt"
import "io"
import "io/ioutil"
import "log"
import "os"

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

type WriterSeeker interface {
    io.Writer
    io.Seeker
}

type APPXWriter struct {
    file WriterSeeker
    err error
}

func NewAPPXWriter(file WriterSeeker) APPXWriter {
    return APPXWriter{
        file: file,
        err: nil,
    }
}

func (w *APPXWriter) Close() error {
    return nil
}

func (w *APPXWriter) u16(data uint16) {
    if w.err != nil { return }
    w.err = binary.Write(w.file, binary.LittleEndian, data)
}

func (w *APPXWriter) u32(data uint32) {
    if w.err != nil { return }
    w.err = binary.Write(w.file, binary.LittleEndian, data)
}

func (w *APPXWriter) bytes(data []byte) {
    if w.err != nil { return }
    _, w.err = w.file.Write(data)
}

func (w *APPXWriter) CreateRaw(header *zip.FileHeader) (io.Writer, error) {
    // Flags:
    const sizeInDataDescriptor uint16 = 0x0008

    fileNameBytes := []byte(header.Name)

    // Write the local file header.
    w.u32(0x04034b50) // local file header signature
    w.u16(0x002d) // version needed to extract
    w.u16(sizeInDataDescriptor) // general purpose bit flag
    w.u16(header.Method) // compression method
    w.u16(header.ModifiedTime) // last mod file time
    w.u16(header.ModifiedDate) // last mod file date
    w.u32(0) // crc-32
    w.u32(0) // compressed size
    w.u32(0) // uncompressed size
    w.u16(uint16(len(fileNameBytes))) // file name length
    w.u16(0) // extra field length
    w.bytes(fileNameBytes) // file name
    // extra field (empty)

    return APPXRawFileWriter{
        w: w,
    }, w.err
}

type APPXRawFileWriter struct {
    w *APPXWriter
}

func (w APPXRawFileWriter) Write(data []byte) (int, error) {
    w.w.bytes(data)
    return len(data), w.w.err
}

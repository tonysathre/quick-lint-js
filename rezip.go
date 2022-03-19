package main

import "archive/zip"
import "bytes"
import "encoding/binary"
import "flag"
import "io"
import "io/ioutil"
import "log"
import "os"
import "hash"
import "hash/crc32"

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
        //fmt.Printf("@@@ %#v\n", zipEntry.FileHeader)

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
    file io.WriteSeeker
    err error

    currentFile *appxWriterFile
}

type appxWriterFile struct {
    name []byte // @@@ not needed?
    dataOffset int64
    uncompressedSize int64
    crc32 hash.Hash32
    realCRC32 uint32
}

func NewAPPXWriter(file io.WriteSeeker) APPXWriter {
    return APPXWriter{
        file: file,
        err: nil,
    }
}

func (w *APPXWriter) Close() error {
    w.writeDataDescriptorIfNeeded()

    return nil
}

func (w *APPXWriter) tell() int64 {
    if w.err != nil {
        return 0
    }
    offset, err := w.file.Seek(0, io.SeekCurrent)
    if err != nil {
        w.err = err
        return 0
    }
    return offset
}

func (w *APPXWriter) writeDataDescriptorIfNeeded() {
    if w.currentFile != nil {
        compressedSize := w.tell() - w.currentFile.dataOffset
        w.u32(0x08074b50) // data descriptor signature
        //@@@ w.u32(w.currentFile.crc32.Sum32()) // crc-32
        w.u32(w.currentFile.realCRC32) // crc-32
        w.u64(uint64(compressedSize)) // compressed size
        w.u64(uint64(w.currentFile.uncompressedSize)) // uncompressed size

        w.currentFile = nil
    }
}

func (w *APPXWriter) u16(data uint16) {
    if w.err != nil { return }
    w.err = binary.Write(w.file, binary.LittleEndian, data)
}

func (w *APPXWriter) u32(data uint32) {
    if w.err != nil { return }
    w.err = binary.Write(w.file, binary.LittleEndian, data)
}

func (w *APPXWriter) u64(data uint64) {
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

    w.writeDataDescriptorIfNeeded()

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

    w.currentFile = &appxWriterFile{
        name: fileNameBytes,
        dataOffset: w.tell(),
        crc32: crc32.NewIEEE(),
        realCRC32: header.CRC32, // @@@
        uncompressedSize: int64(header.UncompressedSize64),
    }

    return APPXRawFileWriter{
        w: w,
    }, w.err
}

type APPXRawFileWriter struct {
    w *APPXWriter
}

func (w APPXRawFileWriter) Write(data []byte) (int, error) {
    w.w.bytes(data)
    w.w.currentFile.crc32.Write(data)
    return len(data), w.w.err
}

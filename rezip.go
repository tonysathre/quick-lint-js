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

    currentFile *appxWriterInProgressFile
    writtenFiles []appxWriterWrittenFile
}

type appxWriterInProgressFile struct {
    dataOffset int64
    uncompressedSize int64
    crc32 hash.Hash32
    realCRC32 uint32
}

type appxWriterWrittenFile struct {
    versionMadeBy uint16
    versionNeededToExtract uint16
    flags uint16
    compressionMethod uint16
    lastModFileTime uint16
    lastModFileDate uint16
    crc32 uint32
    compressedSize int64
    uncompressedSize int64
    fileName []byte
    localHeaderOffset int64
}

func NewAPPXWriter(file io.WriteSeeker) APPXWriter {
    return APPXWriter{
        file: file,
        err: nil,
        currentFile: nil,
        writtenFiles: []appxWriterWrittenFile{},
    }
}

func (w *APPXWriter) Close() error {
    w.writeDataDescriptorIfNeeded()

    for _, file := range w.writtenFiles {
        w.writeCentralDirectoryHeader(&file)
    }

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

// @@@ split.
func (w *APPXWriter) writeDataDescriptorIfNeeded() {
    if w.currentFile != nil {
        writtenFile := &w.writtenFiles[len(w.writtenFiles) - 1]
        writtenFile.compressedSize = w.tell() - w.currentFile.dataOffset
        writtenFile.uncompressedSize = w.currentFile.uncompressedSize
        //@@@ writtenFile.crc32 = w.currentFile.crc32.Sum32()
        writtenFile.crc32 = w.currentFile.realCRC32

        w.u32(0x08074b50) // data descriptor signature
        w.u32(writtenFile.crc32) // crc-32
        w.u64(uint64(writtenFile.compressedSize)) // compressed size
        w.u64(uint64(writtenFile.uncompressedSize)) // uncompressed size

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
    var versionNeededToExtract uint16 = 0x002d
    var versionMadeBy uint16 = 0x002d
    flags := sizeInDataDescriptor
    compressionMethod := header.Method
    lastModFileTime := header.ModifiedTime // @@@
    lastModFileDate := header.ModifiedDate // @@@

    // Write the local file header.
    localHeaderOffset := w.tell()
    w.u32(0x04034b50) // local file header signature
    w.u16(versionNeededToExtract) // version needed to extract
    w.u16(flags) // general purpose bit flag
    w.u16(compressionMethod) // compression method
    w.u16(lastModFileTime) // last mod file time
    w.u16(lastModFileDate) // last mod file date
    w.u32(0) // crc-32
    w.u32(0) // compressed size
    w.u32(0) // uncompressed size
    w.u16(uint16(len(fileNameBytes))) // file name length
    w.u16(0) // extra field length
    w.bytes(fileNameBytes) // file name
    // extra field (empty)

    w.currentFile = &appxWriterInProgressFile{
        dataOffset: w.tell(),
        crc32: crc32.NewIEEE(),
        realCRC32: header.CRC32, // @@@
        uncompressedSize: int64(header.UncompressedSize64),
    }

    w.writtenFiles = append(w.writtenFiles, appxWriterWrittenFile{
        versionMadeBy: versionMadeBy,
        versionNeededToExtract: versionNeededToExtract,
        flags: flags,
        compressionMethod: compressionMethod,
        lastModFileTime: lastModFileTime,
        lastModFileDate: lastModFileDate,
        crc32: 0,
        compressedSize: 0,
        uncompressedSize: 0,
        fileName: fileNameBytes,
        localHeaderOffset: localHeaderOffset,
    })

    return APPXRawFileWriter{
        w: w,
    }, w.err
}

func (w *APPXWriter) writeCentralDirectoryHeader(file *appxWriterWrittenFile) {
    zip64ExtraFieldDataSize := 3*8

    w.u32(0x02014b50)              // central file header signature
    w.u16(file.versionMadeBy)          // version made by                 
    w.u16(file.versionNeededToExtract) // version needed to extract       
    w.u16(file.flags)                  // general purpose bit flag        
    w.u16(file.compressionMethod)      // compression method              
    w.u16(file.lastModFileTime)             // last mod file time              
    w.u16(file.lastModFileDate)             // last mod file date              
    w.u32(file.crc32)                       // crc-32                          
    w.u32(0xffffffff)                  // compressed size (Zip64)
    w.u32(0xffffffff)                  // uncompressed size (Zip64)
    w.u16(uint16(len(file.fileName)))  // file name length                
    w.u16(uint16(2+2+zip64ExtraFieldDataSize))                         // extra field length              
    w.u16(0)                           // file comment length             
    w.u16(0)                           // disk number start               
    w.u16(0)                           // internal file attributes        
    w.u32(0)                           // external file attributes        
    w.u32(0xffffffff)                         // relative offset of local header (Zip64)

    w.bytes(file.fileName) // file name

    // Zip64 extra field
    w.u16(0x0001) // header ID: Zip64
    w.u16(uint16(zip64ExtraFieldDataSize)) // data size
    w.u64(uint64(file.uncompressedSize)) // original size
    w.u64(uint64(file.compressedSize)) // compressed size
    w.u64(0x69696969) // relative header offset
}

type APPXRawFileWriter struct {
    w *APPXWriter
}

func (w APPXRawFileWriter) Write(data []byte) (int, error) {
    w.w.bytes(data)
    w.w.currentFile.crc32.Write(data)
    return len(data), w.w.err
}

package main

import "archive/zip"
import "bytes"
import "encoding/binary"
import "flag"
import "fmt"
import "hash"
import "hash/crc32"
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
        //fmt.Printf("@@@ %#v\n", zipEntry.FileHeader)

        if false {
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
        } else {
            zipEntryFile, err := zipEntry.Open()
            if err != nil {
                log.Fatal(err)
            }
            zipEntry.FileHeader.Method = zip.Store // @@@
            destinationZipEntryFile, err := destinationZipFile.CreateHeader(&zipEntry.FileHeader)
            if err != nil {
                log.Fatal(err)
            }
            _, err = io.Copy(destinationZipEntryFile, zipEntryFile)
            if err != nil {
                log.Fatal(err)
            }
        }
    }
}

type APPXWriter struct {
    file io.WriteSeeker
    err error

    currentFile appxWriterInProgressFile
    files []appxFileInfo
}

type appxWriterInProgressFile interface {
    io.Writer
    finalize(outFile *appxFileInfo)
}

type appxFileInfo struct {
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
        files: []appxFileInfo{},
    }
}

type appxCentralDirectoryInfo struct {
    offset int64
    endOffset int64
}

func (w *APPXWriter) Close() error {
    w.finishFileIfNeeded()

    var centralDirectoryInfo appxCentralDirectoryInfo
    centralDirectoryInfo.offset = w.tell()
    for _, file := range w.files {
        w.writeCentralDirectoryHeader(&file)
    }
    centralDirectoryInfo.endOffset = w.tell()
    w.writeZip64EndOfCentralDirectoryRecord(centralDirectoryInfo)
    w.writeZip64EndOfCentralDirectoryLocator(centralDirectoryInfo)
    w.writeEndOfCentralDirectoryRecord(centralDirectoryInfo)

    return nil
}

func (w *APPXWriter) finishFileIfNeeded() {
    if w.currentFile != nil {
        file := &w.files[len(w.files) - 1]
        w.currentFile.finalize(file)

        w.writeDataDescriptor(*file)

        w.currentFile = nil
    }
}

const zipVersion uint16 = 0x002d

// ZIP flags
const sizeInDataDescriptor uint16 = 0x0008

func (w *APPXWriter) CreateRaw(header *zip.FileHeader) (io.Writer, error) {
    w.finishFileIfNeeded()

    w.files = append(w.files, appxFileInfo{
        flags: sizeInDataDescriptor,
        compressionMethod: header.Method,
        lastModFileTime: header.ModifiedTime, // @@@
        lastModFileDate: header.ModifiedDate, // @@@
        crc32: 0,
        compressedSize: 0,
        uncompressedSize: 0,
        fileName: []byte(header.Name),
        localHeaderOffset: w.tell(),
    })
    w.writeLocalFileHeader(w.files[len(w.files) - 1])

    result := appxRawFileWriter{
        w: w,
        dataOffset: w.tell(),
        crc32: header.CRC32, // @@@ could we compute instead?
        uncompressedSize: int64(header.UncompressedSize64),
    }
    w.currentFile = &result
    return result, w.err
}

// @@@ dedupe
func (w *APPXWriter) CreateHeader(header *zip.FileHeader) (io.Writer, error) {
    w.finishFileIfNeeded()

    w.files = append(w.files, appxFileInfo{
        flags: sizeInDataDescriptor,
        compressionMethod: header.Method,
        lastModFileTime: header.ModifiedTime, // @@@
        lastModFileDate: header.ModifiedDate, // @@@
        crc32: 0,
        compressedSize: 0,
        uncompressedSize: 0,
        fileName: []byte(header.Name),
        localHeaderOffset: w.tell(),
    })
    w.writeLocalFileHeader(w.files[len(w.files) - 1])

    var result appxWriterInProgressFile
    switch header.Method {
    case zip.Store:
        hasher := crc32.NewIEEE()
        result = appxStoringFileWriter{
            w: w,
            dataOffset: w.tell(),
            crc32: &hasher,
        }
    case zip.Deflate:
        result = appxDeflatingFileWriter{
            w: w,
        }
    default:
        return nil, fmt.Errorf("unsupported compression method %#x", header.Method)
    }
    w.currentFile = result
    return result, w.err
}

func (w *APPXWriter) writeLocalFileHeader(file appxFileInfo) {
    w.u32(0x04034b50) // local file header signature
    w.u16(zipVersion) // version needed to extract
    w.u16(file.flags) // general purpose bit flag
    w.u16(file.compressionMethod) // compression method
    w.u16(file.lastModFileTime) // last mod file time
    w.u16(file.lastModFileDate) // last mod file date
    w.u32(0) // crc-32
    w.u32(0) // compressed size
    w.u32(0) // uncompressed size
    w.u16(uint16(len(file.fileName))) // file name length
    w.u16(0) // extra field length
    w.bytes(file.fileName) // file name
    // extra field (empty)
}

func (w *APPXWriter) writeDataDescriptor(file appxFileInfo) {
        w.u32(0x08074b50) // data descriptor signature
        w.u32(file.crc32) // crc-32
        w.u64(uint64(file.compressedSize)) // compressed size
        w.u64(uint64(file.uncompressedSize)) // uncompressed size
    }

func (w *APPXWriter) writeCentralDirectoryHeader(file *appxFileInfo) {
    zip64ExtraFieldDataSize := 3*8

    w.u32(0x02014b50)              // central file header signature
    w.u16(zipVersion)          // version made by                 
    w.u16(zipVersion) // version needed to extract       
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
    w.u64(uint64(file.localHeaderOffset)) // relative header offset
}

func (w *APPXWriter) writeZip64EndOfCentralDirectoryRecord(centralDirectoryInfo appxCentralDirectoryInfo) {
    w.u32(0x06064b50)                  // zip64 end of central dir signature
    w.u64(2+2+4+4+8+8+8+8)                  // size of zip64 end of central directory record                
    w.u16(zipVersion)                  // version made by                 
    w.u16(zipVersion)                  // version needed to extract       
    w.u32(0)                           // number of this disk             
    w.u32(0)                           // number of the disk with the start of the central directory  
    w.u64(uint64(len(w.files))) // total number of entries in the central directory on this disk  
    w.u64(uint64(len(w.files))) // total number of entries in the central directory              
    w.u64(uint64(centralDirectoryInfo.endOffset - centralDirectoryInfo.offset))                  // size of the central directory   
    w.u64(uint64(centralDirectoryInfo.offset))                  // offset of start of central directory with respect to the starting disk number        
}

func (w *APPXWriter) writeZip64EndOfCentralDirectoryLocator(centralDirectoryInfo appxCentralDirectoryInfo) {
      w.u32(0x07064b50)                // zip64 end of central dir locator signature                       
      w.u32(0)                         // number of the disk with the start of the zip64 end of central directory               
      w.u64(uint64(centralDirectoryInfo.endOffset))                // relative offset of the zip64 end of central directory record 
      w.u32(1)                         // total number of disks           
}

func (w *APPXWriter) writeEndOfCentralDirectoryRecord(centralDirectoryInfo appxCentralDirectoryInfo) {
    w.u32(0x06054b50)                  // end of central dir signature    
    w.u16(0xffff)                           // number of this disk             
    w.u16(0xffff)                           // number of the disk with the start of the central directory
    w.u16(0xffff) // total number of entries in the central directory on this disk  
    w.u16(0xffff) // total number of entries in the central directory         
    w.u32(0xffffffff)                  // size of the central directory   
    w.u32(0xffffffff)                  // offset of start of central directory with respect to the starting disk number        
    w.u16(0)                           // .ZIP file comment length       
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

type appxRawFileWriter struct {
    w *APPXWriter
    dataOffset int64
    uncompressedSize int64
    crc32 uint32
}

func (f appxRawFileWriter) finalize(outFile *appxFileInfo) {
    outFile.compressedSize = f.w.tell() - f.dataOffset
    outFile.uncompressedSize = f.uncompressedSize
    outFile.crc32 = f.crc32
}

func (w appxRawFileWriter) Write(data []byte) (int, error) {
    w.w.bytes(data)
    return len(data), w.w.err
}

type appxStoringFileWriter struct {
    w *APPXWriter
    dataOffset int64
    crc32 *hash.Hash32
}

func (f appxStoringFileWriter) finalize(outFile *appxFileInfo) {
    size := f.w.tell() - f.dataOffset
    outFile.compressedSize = size
    outFile.uncompressedSize = size
    outFile.crc32 = (*f.crc32).Sum32()
}

func (w appxStoringFileWriter) Write(data []byte) (int, error) {
    w.w.bytes(data)
    (*w.crc32).Write(data)
    return len(data), w.w.err
}

type appxDeflatingFileWriter struct {
    w *APPXWriter
}

func (f appxDeflatingFileWriter) finalize(outFile *appxFileInfo) {
    // @@@
}

func (w appxDeflatingFileWriter) Write(data []byte) (int, error) {
    w.w.bytes(data)
    return len(data), w.w.err
}

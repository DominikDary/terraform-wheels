package utils

import (
  "archive/tar"
  "archive/zip"
  "bufio"
  "bytes"
  "compress/bzip2"
  "compress/gzip"
  "crypto"
  "crypto/rsa"
  "crypto/sha256"
  "encoding/hex"
  "fmt"
  "gopkg.in/cheggaaa/pb.v1"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "time"
)

/**
 * Private structures, used by the intermediate chain functions
 */
type CloseFunc = func() error
type StreamMeta struct {
  ContentLength   int
  ContentEncoding string
}
type NetworkStreamChain struct {
  Reader io.Reader
  Err    error
  Meta   StreamMeta
  Close  CloseFunc
}

type DownloadFlags int

const (
  WithDefaults       DownloadFlags = 0
  WithoutCompression DownloadFlags = 1
  IgnoreErrors       DownloadFlags = 2
)

/**
 * A customized HTTP client
 */
func getHttpClient(disableCompression bool) *http.Client {
  tr := &http.Transport{
    MaxIdleConns:       10,
    IdleConnTimeout:    30 * time.Second,
    DisableCompression: disableCompression,
  }
  return &http.Client{Transport: tr}
}

/**
 * Start a network stream
 */
func Download(url string, flags DownloadFlags) NetworkStreamChain {
  client := getHttpClient((flags & WithoutCompression) != 0)
  resp, err := client.Get(url)
  if err != nil {
    return NetworkStreamChain{
      nil,
      fmt.Errorf("could not request %s: %s", url, err.Error()),
      StreamMeta{},
      func() error {
        return nil
      },
    }
  }

  // Fail on error resources
  if (flags & IgnoreErrors) == 0 {
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
      return NetworkStreamChain{
        nil,
        fmt.Errorf("server responded with: %s", resp.Status),
        StreamMeta{},
        func() error {
          return resp.Body.Close()
        },
      }
    }
  }

  // Parse Content-Length header
  contentLength, err := strconv.Atoi(resp.Header.Get("Content-Length"))
  if err != nil {
    contentLength = 0
  }

  // Extract content type
  contentEncoding := resp.Header.Get("Content-Encoding")

  // Return a network stream with meta
  return NetworkStreamChain{
    resp.Body,
    nil,
    StreamMeta{
      contentLength,
      contentEncoding,
    },
    func() error {
      return resp.Body.Close()
    },
  }
}

/**
 * Also calculate incoming stream and validate it
 */
func (stream NetworkStreamChain) AndValidateChecksum(checksum string) NetworkStreamChain {
  if stream.Err != nil {
    return stream
  }

  // Split streams, so we can calculate the checksum AND extract
  // while at the same time downloading the file.
  hasher := sha256.New()
  proxyReader := io.TeeReader(stream.Reader, hasher)

  // Return chain
  return NetworkStreamChain{
    proxyReader,
    nil,
    stream.Meta,
    func() error {
      err := stream.Close()
      if err != nil {
        return err
      }

      // Now validate checksum
      csum := hex.EncodeToString(hasher.Sum(nil))
      if csum != checksum {
        return fmt.Errorf("invalid content checksum")
      }

      return nil
    },
  }
}

/**
 * Also validate the signature
 */
func (stream NetworkStreamChain) AndValidatePSSSignature(sig []byte, pub *rsa.PublicKey) NetworkStreamChain {
  if stream.Err != nil {
    return stream
  }

  // Split streams, so we can calculate the checksum AND extract
  // while at the same time downloading the file.
  hasher := sha256.New()
  proxyReader := io.TeeReader(stream.Reader, hasher)

  // Return chain
  return NetworkStreamChain{
    proxyReader,
    nil,
    stream.Meta,
    func() error {
      err := stream.Close()
      if err != nil {
        return err
      }

      // Verify PSS signature
      var opts rsa.PSSOptions
      sum := hasher.Sum(nil)
      err = rsa.VerifyPSS(pub, crypto.SHA256, sum, sig, &opts)
      if err != nil {
        return fmt.Errorf("content signature cannot be verified")
      }

      return nil
    },
  }
}

/**
 * Also show progress as it's downloaded
 */
func (stream NetworkStreamChain) AndShowProgress(prefix string) NetworkStreamChain {
  if stream.Err != nil {
    return stream
  }

  // Create progress bar
  bar := pb.New(stream.Meta.ContentLength).SetUnits(pb.U_BYTES).Prefix(prefix)
  bar.Start()
  proxyReader := bar.NewProxyReader(stream.Reader)

  // Return chain
  return NetworkStreamChain{
    proxyReader,
    nil,
    stream.Meta,
    func() error {
      bar.Finish()
      return stream.Close()
    },
  }
}

/**
 * Also de-compress if the stream has a compressed content-type
 */
func (stream NetworkStreamChain) AndDecompressByContentType() NetworkStreamChain {
  return stream
}

/**
 * Also de-compress if the stream contains compressed data
 */
func (stream NetworkStreamChain) AndDecompressIfCompressed() NetworkStreamChain {
  if stream.Err != nil {
    return stream
  }

  // Peek on the magic header bytes
  bReader := bufio.NewReader(stream.Reader)
  testBytes, err := bReader.Peek(3)
  if err != nil {
    // First close the upstream and then return a detached child with the error
    stream.Close()
    return NetworkStreamChain{
      nil,
      fmt.Errorf("could not peek on the stream: %s", err.Error()),
      stream.Meta,
      func() error {
        return nil
      },
    }
  }

  // If we have a GZip stream, use a GZip reader to de-compress
  // the stream on the fly

  if testBytes[0] == 0x1F && testBytes[1] == 0x8B {
    uncompressedStream, err := gzip.NewReader(bReader)
    if err != nil {
      stream.Close()
      return NetworkStreamChain{
        nil,
        fmt.Errorf("could not open GZip stream: %s", err.Error()),
        stream.Meta,
        func() error {
          return nil
        },
      }
    }

    return NetworkStreamChain{
      uncompressedStream,
      nil,
      stream.Meta,
      func() error {
        return stream.Close()
      },
    }
  }

  // If we have a BZip2 stream, use a BZip2 reader to de-compress
  // the stream on the fly

  if testBytes[0] == 0x42 && testBytes[1] == 0x5A && testBytes[2] == 0x68 {
    uncompressedStream := bzip2.NewReader(bReader)

    return NetworkStreamChain{
      uncompressedStream,
      nil,
      stream.Meta,
      func() error {
        return stream.Close()
      },
    }
  }

  // If we have a plaint-text stream, pass it through
  return NetworkStreamChain{
    bReader,
    nil,
    stream.Meta,
    func() error {
      return stream.Close()
    },
  }
}

/**
 * De-compress the stream on the given directory
 */
func (stream NetworkStreamChain) EventuallyUnzipTo(prefix string, stripComponents int) error {
  if stream.Err != nil {
    return stream.Err
  }

  // Removes `stripComponents` parts from the path given
  applyStrip := func(src string) string {
    parts := strings.Split(src, string(os.PathSeparator))
    if stripComponents >= len(parts) {
      return ""
    }
    return filepath.Join(parts[stripComponents:]...)
  }

  // Read the whole archive in memory (not a good idea for very big files)
  buff := bytes.NewBuffer([]byte{})
  size, err := io.Copy(buff, stream.Reader)
  if err != nil {
    return fmt.Errorf("unzip failed: could not buffer file in memory: %s", err.Error())
  }

  // Oopen the zip stream
  zipReader, err := zip.NewReader(bytes.NewReader(buff.Bytes()), size)
  if err != nil {
    return fmt.Errorf("unzip failed: %s", err.Error())
  }

  // Iterate over the file records
  for _, file := range zipReader.File {
    fName := applyStrip(file.Name)
    fDstPath := filepath.Join(prefix, fName)

    // Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
    if !strings.HasPrefix(fDstPath, filepath.Clean(prefix)+string(os.PathSeparator)) {
      return fmt.Errorf("unzip failed: illegal path: %s", fDstPath)
    }

    if file.FileInfo().IsDir() {
      if err := os.Mkdir(fDstPath, os.ModePerm); err != nil {
        stream.Close()
        return fmt.Errorf("unzip failed: cannot create directory %s: %s", fDstPath, err.Error())
      }
      continue
    }

    // Make File
    if err = os.MkdirAll(filepath.Dir(fDstPath), os.ModePerm); err != nil {
      return fmt.Errorf("unzip failed: cannot create directory %s: %s", filepath.Dir(fDstPath), err.Error())
    }

    outFile, err := os.OpenFile(fDstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
    if err != nil {
      return fmt.Errorf("unzip failed: cannot open %s for writing: %s", fDstPath, err.Error())
    }

    rc, err := file.Open()
    if err != nil {
      return fmt.Errorf("unzip failed: cannot open %s for reading: %s", err.Error())
    }

    _, err = io.Copy(outFile, rc)

    // Close the file without defer to close before next iteration of loop
    outFile.Close()
    rc.Close()

    if err != nil {
      return fmt.Errorf("unzip failed: cannot write %s: %s", fDstPath, err.Error())
    }
  }

  // Close the stream and return any final errors that might have occurred
  return stream.Close()
}

/**
 * De-compress the stream on the given directory
 */
func (stream NetworkStreamChain) EventuallyUntarTo(prefix string, stripComponents int) ([]string, error) {
  var files []string = nil
  if stream.Err != nil {
    return files, stream.Err
  }

  // Removes `stripComponents` parts from the path given
  applyStrip := func(src string) string {
    parts := strings.Split(src, string(os.PathSeparator))
    if stripComponents >= len(parts) {
      return ""
    }
    return filepath.Join(parts[stripComponents:]...)
  }

  // Open the tar stream
  tarReader := tar.NewReader(stream.Reader)
  for true {
    header, err := tarReader.Next()
    if err == io.EOF {
      break
    }

    if err != nil {
      stream.Close()
      return files, fmt.Errorf("untar failed: cannot get next entry: %s", err.Error())
    }

    switch header.Typeflag {

    // Directory
    case tar.TypeDir:
      fName := applyStrip(header.Name)
      if fName != "" {
        if err := os.Mkdir(prefix+fName, 0755); err != nil {
          stream.Close()
          return files, fmt.Errorf("untar failed: cannot create directory: %s", err.Error())
        }
      }

    // File
    case tar.TypeReg:
      fName := applyStrip(header.Name)
      if fName != "" {
        outFile, err := os.Create(prefix + fName)
        if err != nil {
          stream.Close()
          return files, fmt.Errorf("untar failed: cannot create file: %s", err.Error())
        }
        defer outFile.Close()
        if _, err := io.Copy(outFile, tarReader); err != nil {
          stream.Close()
          return files, fmt.Errorf("untar failed: cannot copy file contents: %s", err.Error())
        }

        files = append(files, fName)
      }

    // Other/Unknown
    default:
      // return files, errors.New(
      //   fmt.Sprintf("ExtractTarGz: uknown type: %s in %s",
      //     header.Typeflag,
      //     header.Name))
    }
  }

  // Close the stream and return any final errors that might have occurred
  return files, stream.Close()
}

/**
 * De-compress the stream on the given directory
 */
func (stream NetworkStreamChain) EventuallyUntarOnlyTo(prefix string, filenames []string, stripComponents int) error {
  var extractedFiles int = 0
  if stream.Err != nil {
    return stream.Err
  }

  // Removes `stripComponents` parts from the path given
  applyStrip := func(src string) string {
    parts := strings.Split(src, string(os.PathSeparator))
    if stripComponents >= len(parts) {
      return ""
    }
    return filepath.Join(parts[stripComponents:]...)
  }

  // Open the tar stream
  tarReader := tar.NewReader(stream.Reader)
  for true {
    header, err := tarReader.Next()
    if err == io.EOF {
      break
    }

    if err != nil {
      stream.Close()
      return fmt.Errorf("untar failed: cannot get next entry: %s", err.Error())
    }

    switch header.Typeflag {

    // File
    case tar.TypeReg:
      fName := applyStrip(header.Name)

      // Check if that file should be created
      found := false
      for _, rqName := range filenames {
        if rqName == fName {
          found = true
          break
        }
      }

      if found {
        fDstName := prefix + fName

        if err = os.MkdirAll(filepath.Dir(fDstName), os.ModePerm); err != nil {
          return fmt.Errorf("untar failed: cannot create directory %s: %s", filepath.Dir(fDstName), err.Error())
        }

        outFile, err := os.Create(fDstName)
        if err != nil {
          stream.Close()
          return fmt.Errorf("untar failed: cannot create file: %s", err.Error())
        }
        defer outFile.Close()
        if _, err := io.Copy(outFile, tarReader); err != nil {
          stream.Close()
          return fmt.Errorf("untar failed: cannot copy file contents: %s", err.Error())
        }

        extractedFiles += 1
      }

    // Other/Unknown
    default:
      // return errors.New(
      //   fmt.Sprintf("ExtractTarGz: uknown type: %s in %s",
      //     header.Typeflag,
      //     header.Name))
    }
  }

  if extractedFiles == 0 {
    return fmt.Errorf("Did not find any matching file in the archive")
  }

  // Close the stream and return any final errors that might have occurred
  return stream.Close()
}

/**
 * De-compress the stream on the given directory
 */
func (stream NetworkStreamChain) EventuallyUnzipOnlyTo(prefix string, filenames []string, stripComponents int) error {
  var extractedFiles int = 0
  if stream.Err != nil {
    return stream.Err
  }

  // Removes `stripComponents` parts from the path given
  applyStrip := func(src string) string {
    parts := strings.Split(src, string(os.PathSeparator))
    if stripComponents >= len(parts) {
      return ""
    }
    return filepath.Join(parts[stripComponents:]...)
  }

  // Read the whole archive in memory (not a good idea for very big files)
  buff := bytes.NewBuffer([]byte{})
  size, err := io.Copy(buff, stream.Reader)
  if err != nil {
    return fmt.Errorf("unzip failed: could not buffer file in memory: %s", err.Error())
  }

  // Oopen the zip stream
  zipReader, err := zip.NewReader(bytes.NewReader(buff.Bytes()), size)
  if err != nil {
    return fmt.Errorf("unzip failed: %s", err.Error())
  }

  // Iterate over the file records
  for _, file := range zipReader.File {
    fName := applyStrip(file.Name)

    // Check if that file should be created
    found := false
    for _, rqName := range filenames {
      if rqName == fName {
        found = true
        break
      }
    }

    if !found {
      continue
    }

    // Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
    fDstPath := filepath.Join(prefix, fName)
    if !strings.HasPrefix(fDstPath, filepath.Clean(prefix)+string(os.PathSeparator)) {
      return fmt.Errorf("unzip failed: illegal path: %s", fDstPath)
    }

    if file.FileInfo().IsDir() {
      if err := os.Mkdir(fDstPath, os.ModePerm); err != nil {
        stream.Close()
        return fmt.Errorf("unzip failed: cannot create directory %s: %s", fDstPath, err.Error())
      }
      continue
    }

    // Make File
    if err = os.MkdirAll(filepath.Dir(fDstPath), os.ModePerm); err != nil {
      return fmt.Errorf("unzip failed: cannot create directory %s: %s", filepath.Dir(fDstPath), err.Error())
    }

    outFile, err := os.OpenFile(fDstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
    if err != nil {
      return fmt.Errorf("unzip failed: cannot open %s for writing: %s", fDstPath, err.Error())
    }

    rc, err := file.Open()
    if err != nil {
      return fmt.Errorf("unzip failed: cannot open %s for reading: %s", err.Error())
    }

    _, err = io.Copy(outFile, rc)

    // Close the file without defer to close before next iteration of loop
    outFile.Close()
    rc.Close()

    if err != nil {
      return fmt.Errorf("unzip failed: cannot write %s: %s", fDstPath, err.Error())
    }
    extractedFiles += 1
  }

  if extractedFiles == 0 {
    return fmt.Errorf("Did not find any matching file in the archive")
  }

  // Close the stream and return any final errors that might have occurred
  return stream.Close()
}

/**
 * Write the stream into the designated filename
 */
func (stream NetworkStreamChain) EventuallyWriteTo(filename string) error {
  if stream.Err != nil {
    return stream.Err
  }

  // Download contents to /tool
  f, err := os.Create(filename)
  if err != nil {
    stream.Close()
    return fmt.Errorf("could create destination file: %s", err.Error())
  }
  defer f.Close()

  // Try GZip and fall-back to plaintext
  fileStream := bufio.NewWriter(f)
  if _, err := io.Copy(fileStream, stream.Reader); err != nil {
    stream.Close()
    return fmt.Errorf("could not create file contents: %s", err.Error())
  }
  fileStream.Flush()

  // Close the stream and return any final errors that might have occurred
  return stream.Close()
}

/**
 * Read the stream contents as a byte array
 */
func (stream NetworkStreamChain) EventuallyReadAll() ([]byte, error) {
  if stream.Err != nil {
    return nil, stream.Err
  }

  byt, err := ioutil.ReadAll(stream.Reader)
  if err != nil {
    return nil, err
  }

  err = stream.Close()
  if err != nil {
    return nil, err
  }

  return byt, nil
}

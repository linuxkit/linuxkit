package metadata

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/radu-matei/azure-sdk-for-go/storage"
	"github.com/radu-matei/azure-vhd-utils/upload/progress"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/diskstream"
)

// The key of the page blob metadata collection entry holding VHD metadata as json.
//
const metaDataKey = "diskmetadata"

// MetaData is the type representing metadata associated with an Azure page blob holding the VHD.
// This will be stored as a JSON string in the page blob metadata collection with key 'diskmetadata'.
//
type MetaData struct {
	FileMetaData *FileMetaData `json:"fileMetaData"`
}

// FileMetaData represents the metadata of a VHD file.
//
type FileMetaData struct {
	FileName         string    `json:"fileName"`
	FileSize         int64     `json:"fileSize"`
	VHDSize          int64     `json:"vhdSize"`
	LastModifiedTime time.Time `json:"lastModifiedTime"`
	MD5Hash          []byte    `json:"md5Hash"` // Marshal will encodes []byte as a base64-encoded string
}

// ToJSON returns MetaData as a json string.
//
func (m *MetaData) ToJSON() (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToMap returns the map representation of the MetaData which can be stored in the page blob metadata colleciton
//
func (m *MetaData) ToMap() (map[string]string, error) {
	v, err := m.ToJSON()
	if err != nil {
		return nil, err
	}

	return map[string]string{metaDataKey: v}, nil
}

// NewMetaDataFromLocalVHD creates a MetaData instance that should be associated with the page blob
// holding the VHD. The parameter vhdPath is the path to the local VHD.
//
func NewMetaDataFromLocalVHD(vhdPath string) (*MetaData, error) {
	fileStat, err := getFileStat(vhdPath)
	if err != nil {
		return nil, err
	}

	fileMetaData := &FileMetaData{
		FileName:         fileStat.Name(),
		FileSize:         fileStat.Size(),
		LastModifiedTime: fileStat.ModTime(),
	}

	diskStream, err := diskstream.CreateNewDiskStream(vhdPath)
	if err != nil {
		return nil, err
	}
	defer diskStream.Close()
	fileMetaData.VHDSize = diskStream.GetSize()
	fileMetaData.MD5Hash, err = calculateMD5Hash(diskStream)
	if err != nil {
		return nil, err
	}

	return &MetaData{
		FileMetaData: fileMetaData,
	}, nil
}

// NewMetadataFromBlob returns MetaData instance associated with a Azure page blob, if there is no
// MetaData associated with the blob it returns nil value for MetaData
//
func NewMetadataFromBlob(blobClient storage.BlobStorageClient, containerName, blobName string) (*MetaData, error) {
	allMetadata, err := blobClient.GetBlobMetadata(containerName, blobName)
	if err != nil {
		return nil, fmt.Errorf("NewMetadataFromBlob, failed to fetch blob metadata: %v", err)
	}
	m, ok := allMetadata[metaDataKey]
	if !ok {
		return nil, nil
	}

	b := []byte(m)
	metadata := MetaData{}
	if err := json.Unmarshal(b, &metadata); err != nil {
		return nil, fmt.Errorf("NewMetadataFromBlob, failed to deserialize blob metadata with key %s: %v", metaDataKey, err)
	}
	return &metadata, nil
}

// CompareMetaData compares the MetaData associated with the remote page blob and local VHD file. If both metadata
// are same this method returns an empty error slice else a non-empty error slice with each error describing
// the metadata entry that mismatched.
//
func CompareMetaData(remote, local *MetaData) []error {
	var metadataErrors = make([]error, 0)
	if !bytes.Equal(remote.FileMetaData.MD5Hash, local.FileMetaData.MD5Hash) {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("MD5 hash of VHD file in Azure blob storage (%v) and local VHD file (%v) does not match",
				base64.StdEncoding.EncodeToString(remote.FileMetaData.MD5Hash),
				base64.StdEncoding.EncodeToString(local.FileMetaData.MD5Hash)))
	}

	if remote.FileMetaData.VHDSize != local.FileMetaData.VHDSize {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Logical size of the VHD file in Azure blob storage (%d) and local VHD file (%d) does not match",
				remote.FileMetaData.VHDSize, local.FileMetaData.VHDSize))
	}

	if remote.FileMetaData.FileSize != local.FileMetaData.FileSize {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Size of the VHD file in Azure blob storage (%d) and local VHD file (%d) does not match",
				remote.FileMetaData.FileSize, local.FileMetaData.FileSize))
	}

	if remote.FileMetaData.LastModifiedTime != local.FileMetaData.LastModifiedTime {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Last modified time of the VHD file in Azure blob storage (%v) and local VHD file (%v) does not match",
				remote.FileMetaData.LastModifiedTime, local.FileMetaData.LastModifiedTime))
	}

	if remote.FileMetaData.FileName != local.FileMetaData.FileName {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Full name of the VHD file in Azure blob storage (%s) and local VHD file (%s) does not match",
				remote.FileMetaData.FileName, local.FileMetaData.FileName))
	}

	return metadataErrors
}

// getFileStat returns os.FileInfo of a file.
//
func getFileStat(filePath string) (os.FileInfo, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("fileMetaData.getFileStat: %v", err)
	}
	defer fd.Close()
	return fd.Stat()
}

// calculateMD5Hash compute the MD5 checksum of a disk stream, it writes the compute progress in stdout
// If there is an error in reading file, then the MD5 compute will stop and it return error.
//
func calculateMD5Hash(diskStream *diskstream.DiskStream) ([]byte, error) {
	progressStream := progress.NewReaderWithProgress(diskStream, diskStream.GetSize(), 1*time.Second)
	defer progressStream.Close()

	go func() {
		s := time.Time{}
		fmt.Println("Computing MD5 Checksum..")
		for progressRecord := range progressStream.ProgressChan {
			t := s.Add(progressRecord.RemainingDuration)
			fmt.Printf("\r Completed: %3d%% RemainingTime: %02dh:%02dm:%02ds Throughput: %d MB/sec",
				int(progressRecord.PercentComplete),
				t.Hour(), t.Minute(), t.Second(),
				int(progressRecord.AverageThroughputMbPerSecond),
			)
		}
	}()

	h := md5.New()
	buf := make([]byte, 2097152) // 2 MB staging buffer
	_, err := io.CopyBuffer(h, progressStream, buf)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

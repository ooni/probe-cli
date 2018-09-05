// Code generated by go-bindata. DO NOT EDIT.
// sources:
// data/default-config.json
// data/migrations/1_create_msmt_results.sql

package bindata


import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}


type asset struct {
	bytes []byte
	info  fileInfoEx
}

type fileInfoEx interface {
	os.FileInfo
	MD5Checksum() string
}

type bindataFileInfo struct {
	name        string
	size        int64
	mode        os.FileMode
	modTime     time.Time
	md5checksum string
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) MD5Checksum() string {
	return fi.md5checksum
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _bindataDataDefaultconfigjson = []byte(
	"\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x94\x41\x6f\xdb\x3c\x0c\x86\xef\xf9\x15\x82\xce\x75\x53\xe0\xbb\xe5" +
	"\xf8\xdd\x76\xd8\x3a\x60\xbb\x15\x85\x20\x5b\xb4\x4d\x4c\x26\x35\x91\x4e\x16\x0c\xfd\xef\x83\xdc\x24\x56\xda\xae" +
	"\xeb\xd1\xef\x4b\x53\xe2\x43\x52\xbf\x37\xc6\x58\x67\x77\xc6\x7e\x1f\x51\x0c\x8a\x39\xf2\x9c\xcd\xfd\xfd\x97\x4f" +
	"\xe6\x6b\xe6\x16\x4c\xc7\xd4\xe3\x60\x7a\x8c\x70\x6b\xbe\x01\x98\x51\x35\xc9\x6e\xbb\x65\x26\xbc\x45\xde\x8e\x10" +
	"\xd3\x36\x95\xd8\xa6\x8b\x68\x7a\xce\xa6\x48\xf6\x66\x49\xbd\x87\x2c\xc8\x64\x77\xe6\xee\x59\x40\xea\x39\x4f\x10" +
	"\x5c\xc7\x24\x40\x6a\x77\xa6\xf7\x51\xe0\xe4\x8a\x6b\x41\xbd\xdd\x19\xcd\xf3\xb3\xe6\x67\x65\x37\xa7\xe0\x15\x6a" +
	"\x59\x46\x9f\x91\x06\xbb\x33\xa5\x06\x63\x2c\x52\x17\xe7\x00\x0e\x53\x9d\xb2\x32\xbc\x50\x95\xa0\x32\x86\x24\xd7" +
	"\xc6\x9c\x22\xfb\xe0\x32\xc8\x1c\xf5\xec\x6d\x8c\x79\x5a\x4e\x26\x56\xec\xb1\xf3\x8a\x4c\xb2\x9e\x0f\xe4\xdb\x08" +
	"\xe1\x3a\xd3\x12\x7b\x74\x4c\x4e\x41\xd4\x75\x3c\xa5\x08\xfa\x0c\xe4\xcd\x30\x82\x83\x9c\xef\x7f\x39\xb1\x20\x98" +
	"\xbc\x42\x58\xb2\x5c\x55\xbd\x9e\x5a\x97\x7c\x52\x97\xf0\x92\xee\x61\x91\x8d\xb1\x07\x68\x9b\x8e\x89\xa0\x53\xdc" +
	"\xa3\x1e\xed\xcd\xd9\xe9\x7d\x07\x2d\xf3\x8f\x66\x02\x11\xa0\x01\xf2\xea\x1d\x46\xaf\xe2\x53\x5a\x15\x85\x08\x43" +
	"\xf6\xd3\xaa\x04\x2f\xe3\xfa\x45\x41\xd7\x8f\x32\x31\x0d\xd2\xde\x47\x0c\x4d\x86\x9f\x33\x88\x36\x11\x09\x5e\x84" +
	"\x8c\xe0\x03\xe4\xa6\x47\x88\xa1\x99\x3c\x61\x9a\xe3\x42\xd9\x2e\x61\x8f\xa7\xe2\x26\x26\x1d\xe3\xd1\xf9\x18\xf9" +
	"\xe0\xa9\x2b\x63\x61\xff\xbb\xbb\xfb\xfc\xbf\xbd\x10\x5b\x68\x0b\x68\x81\x55\xf5\xe8\x00\xad\xa0\xc2\xaa\x54\xac" +
	"\x3a\xaf\x30\x70\xc6\xc5\x7d\x78\x5c\xec\xa7\xcb\xa4\x88\x7a\x52\x57\xd8\xf8\xa1\x6e\xc0\x3b\xb0\xdf\x87\xfa\x16" +
	"\xd6\x1a\xec\x49\xba\xbe\x47\x82\x5c\xb6\xe7\x54\xf4\x47\x6e\x50\x1a\x71\x4e\x55\x77\xc7\x09\xe4\x3d\xe4\x82\xae" +
	"\x4c\x97\x7d\xc3\x73\x89\xb3\xbe\x0e\x28\x8d\xfe\xeb\xdf\x95\x79\xfd\xfb\x55\x19\x13\x86\x10\xa1\xe5\x5f\x1f\x2c" +
	"\xe2\xdf\x03\xf4\xc1\x11\xba\xf0\x5c\x57\x2b\xec\x0b\xcd\xf0\xfa\x1d\xe9\x78\x26\xcd\xc7\x17\x2f\x83\x80\x0b\x3c" +
	"\x79\x24\xd7\x67\xa6\xd3\x2e\xd6\xab\x27\x40\xc1\x75\xb9\x70\xc8\x50\x10\xd4\xef\xc7\xe6\x69\xf3\x27\x00\x00\xff" +
	"\xff\x42\x02\xc0\xed\x72\x05\x00\x00")

func bindataDataDefaultconfigjsonBytes() ([]byte, error) {
	return bindataRead(
		_bindataDataDefaultconfigjson,
		"data/default-config.json",
	)
}



func bindataDataDefaultconfigjson() (*asset, error) {
	bytes, err := bindataDataDefaultconfigjsonBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{
		name: "data/default-config.json",
		size: 0,
		md5checksum: "",
		mode: os.FileMode(0),
		modTime: time.Unix(0, 0),
	}

	a := &asset{bytes: bytes, info: info}

	return a, nil
}

var _bindataDataMigrations1createmsmtresultssql = []byte(
	"\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x59\x6d\x6f\xdb\x46\xf2\x7f\xef\x4f\x31\x30\x8a\xd6\xc6\x5f\x92\x93" +
	"\xfc\xd3\xe0\xce\xd7\xa2\x70\x13\x25\xa7\x36\x96\x03\x59\xbe\x26\x38\x1c\xc4\x15\x39\x94\xb6\x5e\xee\x32\xfb\x20" +
	"\x46\xf7\xe9\x0f\x33\xbb\xa4\x48\xe5\xc9\x01\x9a\x17\x8e\x44\xee\xce\xce\xe3\x6f\x7e\xb3\x1a\x8f\xe1\xff\x2a\xb9" +
	"\xb1\xc2\x23\xbc\x30\x8d\x3e\xe9\x3f\xb8\xf5\xc2\x63\x85\xda\xff\x8a\x1b\xa9\x4f\x4e\x5e\x2c\x6e\xde\xc0\xf2\xea" +
	"\xd7\xd7\x53\xc8\x2c\xba\xa0\xbc\xcb\xfe\x31\x78\x5a\xa1\x70\xc1\xf2\x9e\xe3\x57\xc1\xaa\xe3\x47\x1a\x7d\x63\xec" +
	"\x3d\x3d\xfe\xf4\xb9\x53\x5d\x0c\xdf\xdc\xd5\x5f\x54\xf0\xf9\x62\x7a\xb5\x9c\x0e\x4e\x84\xb3\x13\x80\x4c\x16\x19" +
	"\xcc\xe6\xcb\xe9\xab\xe9\x02\xde\x2c\x66\xd7\x57\x8b\x77\xf0\xfb\xf4\x1d\x5c\xdd\x2d\x6f\x66\xf3\xe7\x8b\xe9\xf5" +
	"\x74\xbe\x1c\xd1\xca\x60\x55\x06\xff\xba\x5a\x3c\xff\xe7\xd5\xe2\xec\xc9\x8f\x3f\x9e\xc3\xfc\x66\x09\xf3\xbb\xd7" +
	"\xaf\x47\x30\x1e\xc3\xdb\xb7\x6f\x41\x3a\xf0\x5b\xe9\x40\x19\xbd\x01\xd4\x26\x6c\xb6\xbf\xd0\xd6\x5c\x78\xdc\x18" +
	"\xbb\x5f\xe5\xa6\xc0\x83\x90\x63\x11\xcb\x2d\x42\x2e\xbd\xfc\x2f\x6a\x25\xd6\xd0\xee\x02\xda\x05\xa5\xb1\xe0\xb7" +
	"\x78\x02\x0f\xfb\x37\x1e\x83\x93\x1e\x27\xf0\x07\x42\x70\x48\x5b\xc1\x79\x2b\xf5\x06\xe6\x37\xf3\x29\x78\x03\x05" +
	"\x6a\xe3\xbf\x45\xa0\x36\x70\xaf\x4d\xa3\x87\x9a\x4d\x4e\xd8\x44\x13\xb4\xff\xc8\xc2\x27\x07\x0b\x5b\x03\x7d\x63" +
	"\x40\xa1\xf7\x68\x21\xed\x89\xf6\x35\x5b\x99\x6f\xd9\x7d\x0f\xd3\x68\x3c\x86\xbb\xc5\x6b\x58\x23\x39\xdb\x81\x37" +
	"\x27\xe7\x31\x59\xfe\x40\xc8\x2d\x52\x12\x08\x70\x58\x0b\xce\x07\x2f\xd6\x2a\xfa\xb0\x4d\x2d\xfe\xf2\x04\x2c\x0a" +
	"\x67\xb4\xbb\xa4\x9d\x8f\x27\xf0\xd2\x58\x70\xa6\x42\x30\x25\xbb\x6c\x27\xb1\x71\xd0\x6c\xd1\x22\x68\xc4\x82\x1f" +
	"\x7a\xe3\x85\x02\x1d\xaa\x35\x5a\x5a\x98\x72\xbb\xe8\x64\x8f\x48\x9a\xf4\x3f\x38\xd8\x18\xf2\xb8\x37\xb0\x46\xa8" +
	"\x42\xbe\x85\xca\x58\x04\x2c\x4b\x99\x4b\xd4\x9e\xde\xfc\x19\x9c\x07\x65\xcc\x7d\xa8\x59\x3a\x7b\x85\xc4\x5a\xd3" +
	"\x38\x90\x3a\xfa\x64\x3c\x8e\x36\x4c\xe8\xd3\x93\x09\x9c\x55\xc6\x79\x90\x55\x6d\xac\x17\xda\x9f\x93\xd9\x8d\x88" +
	"\x12\xc5\xce\xc8\x02\x8a\x50\x2b\x99\x0b\x4f\x0a\x08\x58\x07\x9d\x6f\x49\xaa\xd4\xa5\xb1\x95\xf0\xd2\x90\x64\xe1" +
	"\x59\xd5\xa1\xa2\xb9\xa9\x2a\x7a\x6b\xc0\xe1\x0e\x2d\xd9\xda\x3a\x8d\x14\x0c\x0e\x2d\x6d\x31\x9a\x95\x99\x7e\x10" +
	"\x55\xad\xf0\x32\xf9\xbe\x12\x7b\x68\xa4\xdb\xb2\x22\x45\x41\xff\x71\x4d\xc4\x08\xd0\x7e\x65\xf2\x78\x7c\x69\x4d" +
	"\xd5\x3a\xba\xb6\x66\x8d\xf1\x09\x7d\x7d\xf5\xe6\x96\xe4\x19\xcb\x32\x5c\xa8\xc9\x4e\x0e\x99\x50\xca\x34\xac\x6b" +
	"\xab\x8a\x37\x70\x9a\x1b\x6b\x31\xf7\xa7\x20\xa0\x92\x2e\x57\xc2\x39\x59\x4a\x2c\xa0\x87\x3b\x49\x60\x21\x1d\xf9" +
	"\x24\x48\xb7\x25\x31\x6b\xf4\x0d\xa2\x86\x46\x96\x12\x84\x2e\xa0\x32\x6b\x49\x7e\x1e\x42\x46\x87\x48\xdf\x0a\x1b" +
	"\x69\xe3\x4a\x8b\x0a\x87\xf8\xc1\x35\x7f\x1b\x6b\x92\xde\x82\xc5\xda\xa2\x43\xed\x5b\xf3\xfa\x7b\x53\x81\xac\xf7" +
	"\x50\x60\x29\x82\xf2\x14\x82\xda\xd4\x41\x09\x8f\x05\xac\x85\xc3\xe2\x6b\x95\x43\x0e\xd0\x2c\xf9\xea\x76\x3e\x79" +
	"\xc0\xea\x04\x1e\xbd\x42\xba\xc7\x3d\x39\xdc\x62\x89\x16\x75\x1e\x23\x9a\x32\xf5\x01\x02\x0f\xa9\xe0\x46\xb0\xc6" +
	"\x5c\x90\xf8\x66\x98\x35\xa7\xa8\xad\xcc\xb7\xa7\x0f\x15\xd7\x48\x9f\xea\xaa\x10\x5e\xc4\x8a\x41\x28\x83\x0f\x16" +
	"\x27\xfd\x10\xf8\x7d\xdd\x0b\xc1\xe3\x67\x31\x02\x37\x9a\xab\x9d\xe2\x3f\x4a\xc1\x27\x44\x83\x4c\xd6\x87\xc5\x4f" +
	"\x1f\xf5\xb1\x3a\x06\xce\x58\x74\xe4\x9a\x18\xc1\x2e\x78\x31\xb7\x4d\x09\x42\x83\xac\x77\x4f\x29\xe7\x64\xbd\x7b" +
	"\x46\x99\x6c\xd1\xb9\x87\xf8\x7d\xc9\x65\xa2\x37\x48\x35\x5e\x53\xa4\xa3\xb0\x4e\x08\x28\x79\x8f\x97\x0f\x90\xf4" +
	"\xe8\xd1\xa3\x47\x97\x5f\xff\x33\x7a\x80\xa8\x98\x80\xd2\xc1\xff\xff\x1d\xf2\xad\xb0\x6c\x49\x26\x9c\xe6\x52\x38" +
	"\x7b\xda\xf3\xd0\x5f\xd1\x11\x18\xcf\x87\x45\xd8\x52\x0b\xae\xc1\x6f\xa9\xc2\xe4\x54\xe9\x20\x17\x9a\x20\xce\xc4" +
	"\xa0\x9f\x36\xb8\xa6\x36\xe9\x4e\x47\x70\x2a\x2b\xfa\x5b\xa3\x65\x80\xd4\x39\xd2\xd7\x4a\x16\x85\xc2\xb5\xf9\x70" +
	"\x1a\xe3\x96\x79\x74\x7e\xb5\xb1\x26\xd4\x47\x25\xfd\xf8\xd9\xd0\x01\xc3\x02\x2a\x64\xc9\x15\xe3\xc1\x79\x61\xfd" +
	"\xca\xcb\x0a\x19\x6e\x6c\xd0\xf4\x79\x50\x0d\x1d\x90\x2b\x67\x60\x2b\x76\xd8\x8a\xe3\x04\xf7\xa6\x45\x35\x4e\x74" +
	"\xb3\x43\xbb\x45\x51\x90\x3d\xdc\xf8\x22\xe0\x5b\x64\xc8\xa4\x23\x8c\xdf\xa2\x85\x52\xe4\xde\x58\x17\x41\x3f\xc9" +
	"\xdb\x18\x90\x9a\x11\x1a\x81\x0c\x9b\x1c\x7c\x25\x18\x60\xa8\x07\x88\xfd\x25\x64\xb7\x77\xd7\x67\x49\xd5\x73\x78" +
	"\xb9\xb8\xb9\x86\x01\xa3\x83\x46\x2a\x05\x42\x35\x62\xef\xc8\xbf\x3f\xfd\xdc\x4a\xca\xd2\xae\xb8\xe9\x10\x41\xee" +
	"\x5f\xf4\xc2\xc1\x4f\xe7\xd1\xb5\x07\xcf\x64\xf0\xe2\x6a\x39\x5d\xce\xae\xa7\x47\x2e\x6d\xa5\x65\xb0\x98\x5e\xbd" +
	"\xee\xbd\x6c\x8f\xbb\x73\xc8\x3d\x47\xea\x82\x9a\x1f\x82\x2c\x0f\x9d\x62\x2b\x1c\x38\x02\x7b\xc6\x8d\xa8\x4b\xca" +
	"\x24\xb7\xa2\x56\x8f\x45\x06\xcb\xd9\xfc\x1d\xe5\xf3\xe3\xf3\x4f\x88\xe7\x1c\xa2\x72\x84\x52\x89\x0d\x49\xfd\xe4" +
	"\x69\x51\x34\x2d\x2c\x38\xd3\xb8\x5f\xe6\xc1\x52\x02\xa8\x3d\xc5\x5c\x4b\xbd\x99\x74\x67\xd3\xaa\xcf\x9c\xcc\x4b" +
	"\x28\xee\xab\xe0\xc4\x06\x57\xa1\x3e\xe4\xfc\xe7\x57\x15\xa6\xd1\x9f\x5b\x37\x1e\xc3\x8c\xb8\x09\xb5\x5c\xb1\x26" +
	"\x75\x98\x03\xc5\xfe\x4c\x3d\xdf\xb3\x0d\x95\xf8\x20\xab\x50\x81\x42\xbd\xf1\x0c\xcc\x4f\x9e\x3d\x02\x91\x28\x2e" +
	"\x53\xdd\x2e\x2f\x8f\xd6\x9a\x12\x4a\xa9\x10\x6a\xe1\xb7\xc4\x13\xa0\x91\xba\x30\x4d\x82\xbe\x4c\x99\xcd\x8a\xde" +
	"\xaf\xe8\x7d\x0f\x1a\x9e\xf5\x40\xf6\x13\xd5\x3f\x4c\xb8\xbf\x08\x02\x2e\xdb\x57\xa5\xc8\x71\x6d\xcc\xfd\xaa\x42" +
	"\xe7\x50\x6f\xd0\xb6\x6f\x3c\x2a\xdc\x58\x51\x9d\x74\x38\x28\xbc\x13\x75\xdd\x7e\xdf\x7a\x5f\xaf\xa8\x02\xd1\xae" +
	"\x4a\x89\xaa\x58\x55\x42\x4b\x6e\xcc\xd2\xe8\xc1\x2a\xa9\x77\x42\xc9\x62\x65\xf1\x7d\x20\x1c\x51\x52\xf7\x6a\xdb" +
	"\x6d\xdb\xcf\xba\xf0\x3d\xb4\x19\xe2\xcc\xb3\xa7\x1f\xa5\xc7\x5f\x51\x38\x2f\xe3\x7c\x01\x75\xb0\xb5\x71\x8c\x8e" +
	"\x89\x5d\xb4\x6c\x24\x52\xb4\x3e\x7f\x4c\xad\x36\x15\x75\x2b\x89\x39\xf3\x08\xf6\x26\x80\xdb\x9a\xa0\x0a\xa8\x65" +
	"\x7e\x1f\x9b\xb2\xb4\xce\xf7\x91\xa3\x15\xf1\xdb\xcd\x6c\x0e\xce\x58\xa6\x32\xfb\x56\xd2\xc1\xae\x0e\x98\xde\x99" +
	"\x40\x35\xf5\x83\x67\x5c\xe4\xbd\x9b\x20\xac\xd0\x1e\x91\xa1\x0d\x88\xb2\xee\xe1\x4c\xd6\x23\x10\x4e\x8f\xda\x9e" +
	"\x32\x1a\xb0\xa9\xf3\x56\x5e\xcc\x63\x70\xc4\xb0\xa4\x06\x01\xa7\x7d\xed\x1c\x12\xa5\x74\xce\xe4\x92\x59\x16\x61" +
	"\x32\x9c\x46\x7b\xdb\x86\xd0\x8a\xed\x67\xe2\xc7\xee\x9d\x1b\x1f\xe7\x8f\x8d\x51\x42\x6f\x2e\x09\xe6\x5b\xf4\x60" +
	"\x4b\x1c\x4d\xac\xbd\xd6\x94\x45\x4c\x20\xfc\xce\x44\xee\xe5\x0e\xb3\x11\x38\x73\xd2\x67\x3e\xd2\x01\xbe\x0f\x72" +
	"\x27\x54\x9a\x25\x18\x6d\xd6\xc8\x11\xb3\x81\x81\xa7\x14\xca\x1d\xdc\x97\xf1\x31\x19\x2c\xa7\x6f\x53\x55\x3c\x00" +
	"\x7e\x52\x9f\x8e\x30\xd1\x29\x2c\xa0\xc0\x88\x7a\x05\x48\xb7\x0a\xb5\x32\xa2\xc0\x82\x81\x71\x04\x52\x3b\x9f\x9a" +
	"\x12\x0f\x38\xc1\x49\xbd\x39\x38\x3d\x2d\x5f\x95\x42\x2a\x2c\x46\x31\x0c\xc2\xb7\x54\x50\x9b\x14\xdf\x4e\x2a\x23" +
	"\x52\x2f\x32\x45\xe8\x0a\x87\x83\xe2\xd0\xfb\x01\xa4\xb6\x3b\x1f\x08\xe8\xc7\xf2\xa3\x62\x4c\x75\x83\xe6\x28\x74" +
	"\x5d\x24\x25\x35\x85\x8a\x7a\xba\xe5\x65\xad\x3c\x8b\x63\xda\x20\xfd\x41\x93\x28\xea\x4b\xf0\x4e\x2b\x82\xc5\x55" +
	"\xe5\x36\x47\x23\xc2\xc9\x91\x3d\x0f\x10\xd6\x5b\xf8\x25\x99\xd4\x05\xdc\xc7\x0d\x8c\xa3\xc0\xc9\x55\x0b\xeb\x65" +
	"\x1e\x94\xb0\x03\xc7\x50\x0f\x5d\x53\x0f\x4d\x96\x0a\x5d\x1c\x72\x12\x2d\x96\x26\xf1\x92\xbb\x19\x23\xad\x17\xf7" +
	"\x98\xb2\x95\x98\x86\xc8\xe3\x7c\xeb\x0d\xa0\x64\x5e\xb2\x95\x05\x82\xf4\xdd\xec\x77\xf0\x24\xf7\x50\x42\x13\x9e" +
	"\x03\x63\x57\xe2\xe2\x56\x28\x9c\xa7\x41\xae\x9b\x29\xc5\x5a\x2a\xe9\xd3\x68\x32\x88\x40\xba\x9a\x29\x0c\xe5\x16" +
	"\x13\xaa\x96\x5d\xa5\x2c\xee\x8d\x32\x26\xc1\x19\x0b\xe8\x19\xfd\x4b\x17\x05\x8b\x36\xe8\x6f\x48\x29\x87\x76\x87" +
	"\x76\xec\xc8\xc6\xc8\xc8\x56\xb2\x00\x8b\x3e\x58\xcd\x50\x97\x46\x7e\xa5\x90\xd8\xd9\x04\x7e\xdd\x0f\x4b\xe5\xb0" +
	"\xe9\x7b\x90\xba\x0e\x3e\x02\x2b\x79\xf6\x7d\x20\x5f\xb0\xf5\xb5\x24\xe5\x4b\xf4\xe9\x0a\xa5\xaf\x7c\xe7\x86\xe9" +
	"\x87\xee\xe3\xab\xe9\x92\x1b\x92\xbb\xbc\xb8\x10\xb5\x9c\x18\xa3\xe5\x44\x1a\xfa\x7c\xb1\x7b\x7c\xd1\xef\xb4\xbf" +
	"\xf0\xa9\x3f\x7f\x37\x9b\xbf\xb9\x5b\x7e\xdf\xa9\xf3\xf3\x77\x8b\xe9\x9b\x9b\xc5\x72\x35\x7b\x71\x90\xef\xad\xc8" +
	"\x7d\x0f\xe8\xa5\xc7\xea\x30\xd3\x27\xfa\xfe\xef\xff\x64\xa0\xa4\xf3\x6d\x51\xe9\xa8\x77\xd7\x88\xfb\x7d\x7e\xc5" +
	"\x97\x6e\xde\xc0\x26\x91\x92\xdf\x6e\x6f\xe6\xf1\xca\x60\x68\x24\x8d\xa0\x3d\xf2\x8a\x2e\x8e\x15\x3b\xa1\x02\x3a" +
	"\x38\xcb\x3a\xbd\xb3\x11\x64\x6c\x51\x76\x0e\xc2\x72\x45\x97\x41\x1d\xbc\x27\x3a\x4a\xd3\x13\xce\x45\x41\x89\x2f" +
	"\x94\x45\x51\xec\x63\x01\xd4\xd6\xe4\xc4\x15\xba\x30\xd6\xb2\x46\xea\xe8\xa3\x1e\x1e\xc8\xaa\x56\x51\x48\xae\x50" +
	"\xe8\x50\xf3\x64\x98\xc4\x74\xe8\xd6\x77\x78\x02\x8e\x83\xc6\x1f\x5f\x1a\xf4\x69\x0c\x4f\x52\x0d\xb9\x51\x9b\x96" +
	"\xf4\x33\xf9\x6a\x0b\xf5\x2b\x93\xdd\x78\x9c\xae\xcb\x8a\x49\x02\x9b\x60\xd5\x57\x9a\x59\x9b\xe1\x04\xd3\x7b\xf4" +
	"\xc4\x86\x51\xd0\xb8\xdd\xde\xe6\x74\x09\x3d\x82\x75\x60\x54\x27\x5f\xd7\x4a\x30\xef\x4d\x57\x43\x83\x56\x26\x7c" +
	"\xbc\x77\xab\x8d\x3c\xb0\x02\x8d\xc2\xf6\x06\xf9\x38\x77\x23\x5e\x76\xb9\xbb\x91\x7e\x1b\xd6\x93\xdc\x54\x17\x94" +
	"\xc2\x17\x6d\x04\x2e\xd6\xca\xac\x2f\x2a\xe1\x3c\xda\x8b\xc2\xe4\x8e\x5f\x8f\x43\x90\xc5\xa4\x2a\xe0\xfb\x3e\x29" +
	"\xfb\xa2\x1c\xe9\x5c\x40\x77\xf1\xf4\x6f\xd1\x35\xfd\xd4\x4c\x2e\x22\x3e\x76\xec\x99\x04\xa6\xae\xb5\x23\x17\x91" +
	"\x50\x09\x68\xe7\x4d\x9e\xb6\x46\x31\xb1\x04\xdf\xd4\x92\x3f\x69\xa8\x57\x1d\xfb\x59\x2b\x93\xdf\x53\x73\xa4\x2e" +
	"\x4e\x08\xa8\x61\x76\xcd\x1b\xdb\x31\x23\x7d\x75\x34\xa3\xb9\x84\x04\xf5\x97\x05\xc9\x92\xaf\xc8\xd2\x50\x0b\x8d" +
	"\x70\x50\xa0\xc7\x9c\xe3\xdf\xe3\x58\x94\x5d\x19\xb1\xb2\x0c\x04\x64\xcf\x6f\xee\xe6\xcb\xb3\xf3\xac\x2b\x3d\x2e" +
	"\xac\x23\xfe\x17\xa1\x3a\x15\xab\xe8\xee\x31\x8f\xb4\x80\x68\xbf\xb1\xdd\x83\xd9\x35\xa9\xed\x3a\x8c\x15\xda\x54" +
	"\x42\xed\xfb\x28\xfb\x89\x01\x4c\x83\xa9\xc5\xfb\x90\x20\xc1\x79\x1b\x72\xca\x93\x51\xba\xac\x6d\x88\x51\x51\x2b" +
	"\xea\xdf\xe6\x32\x9b\xbe\xc7\x7d\x47\x55\x9b\x74\xab\x9b\x2e\xd7\x87\x0c\x03\xbd\x90\xca\xa5\x2b\x60\x02\x2b\x16" +
	"\xd5\x6b\x4b\x0e\xce\xf0\xc3\xa4\xdf\xb3\x62\x41\x5f\xd0\xf4\x45\x1f\xc0\xd5\x24\xdd\x94\x30\x7f\xb1\x1c\x25\x5f" +
	"\x31\x89\x2a\x5b\xfb\xa9\x1c\x38\x33\xc8\x2d\x1d\xdd\x42\x9f\x4f\xce\x7b\x13\x00\xe9\x9c\xb1\xa5\x7d\x4f\x20\xe4" +
	"\xd6\xb8\xf6\x6a\x75\xd0\xc7\x98\x4f\xfb\x74\xb9\x12\xef\xdb\xc0\x9b\x0d\x52\xc7\xed\x00\x86\x0c\xf9\x5c\xa5\x7f" +
	"\x3c\xf0\xee\x84\x95\x7c\x10\x73\x06\xa9\x3d\x5a\x2d\x94\xe2\x9e\x4b\xc0\x1f\x19\x3e\x8d\x74\x6d\x23\x35\x7a\x5c" +
	"\x48\x77\xff\x09\x44\x75\x93\x3f\x9d\xd1\x13\x98\x79\xa6\x7b\x15\x71\x04\x87\xda\xb1\xee\x8d\xa5\x72\x20\x26\x1b" +
	"\x87\x48\xb4\x80\x7c\x2b\x74\x18\x0c\xb6\xc6\xb0\xe7\xae\x7f\xe7\xc0\xd4\x16\x77\xe9\xda\xb4\x25\x12\x24\xa4\x85" +
	"\x9a\x28\xc7\x68\x62\x0c\xf7\xe9\x1a\xab\x12\x07\x61\xc4\x03\x2a\xa1\xf7\x03\x0d\xf9\xdc\x92\x6f\x82\xfb\x78\xfc" +
	"\xb5\xb9\x35\xc5\xe7\xe5\xcd\x62\x3a\x7b\x35\xe7\x51\xf4\xac\xe7\xea\x73\x58\x4c\x5f\x4e\x17\xd3\xf9\xf3\xe9\xed" +
	"\xe1\x3e\xeb\x8c\xc6\xd8\xf3\x04\xd4\x37\x73\x78\x31\x7d\x3d\x5d\x4e\xe1\xf9\xd5\xed\xf3\xab\x17\x53\x7a\x72\xf7" +
	"\x86\xe6\xba\xf6\x09\x37\x81\x59\x49\xe9\x5b\xa0\x42\x1f\x69\x0c\xe7\x65\x9f\xe4\x3c\xf4\xa7\x9d\xe4\x07\xa1\xd4" +
	"\x71\x11\xb8\xf4\x8b\x40\x3c\xa5\xa0\xe9\xbf\x41\xa5\x26\x9f\xb0\x31\x75\x8d\xa1\x81\xfc\x43\x5b\xb4\x6e\x74\xbc" +
	"\xe7\xac\x3f\x39\x0d\xb7\xf5\x2e\xdb\xa3\x67\xce\xbf\xf4\x3b\xe0\xff\x02\x00\x00\xff\xff\x29\x5f\x48\x5d\xaa\x1c" +
	"\x00\x00")

func bindataDataMigrations1createmsmtresultssqlBytes() ([]byte, error) {
	return bindataRead(
		_bindataDataMigrations1createmsmtresultssql,
		"data/migrations/1_create_msmt_results.sql",
	)
}



func bindataDataMigrations1createmsmtresultssql() (*asset, error) {
	bytes, err := bindataDataMigrations1createmsmtresultssqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{
		name: "data/migrations/1_create_msmt_results.sql",
		size: 0,
		md5checksum: "",
		mode: os.FileMode(0),
		modTime: time.Unix(0, 0),
	}

	a := &asset{bytes: bytes, info: info}

	return a, nil
}


//
// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
//
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
// nolint: deadcode
//
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

//
// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or could not be loaded.
//
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// AssetNames returns the names of the assets.
// nolint: deadcode
//
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

//
// _bindata is a table, holding each asset generator, mapped to its name.
//
var _bindata = map[string]func() (*asset, error){
	"data/default-config.json":                  bindataDataDefaultconfigjson,
	"data/migrations/1_create_msmt_results.sql": bindataDataMigrations1createmsmtresultssql,
}

//
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
//
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, &os.PathError{
					Op: "open",
					Path: name,
					Err: os.ErrNotExist,
				}
			}
		}
	}
	if node.Func != nil {
		return nil, &os.PathError{
			Op: "open",
			Path: name,
			Err: os.ErrNotExist,
		}
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}


type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{Func: nil, Children: map[string]*bintree{
	"data": {Func: nil, Children: map[string]*bintree{
		"default-config.json": {Func: bindataDataDefaultconfigjson, Children: map[string]*bintree{}},
		"migrations": {Func: nil, Children: map[string]*bintree{
			"1_create_msmt_results.sql": {Func: bindataDataMigrations1createmsmtresultssql, Children: map[string]*bintree{}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

package smb

import (
	"io"
	"os"

	"github.com/oiweiwei/go-smb2.fork"
)

// shareOpenFile and shareRemoveFile are package-level vars to allow unit tests to
// stub SMB share interactions without requiring a live SMB server.
var (
	shareOpenFile = func(share *smb2.Share, name string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		if share == nil {
			return nil, os.ErrInvalid
		}
		f, err := share.OpenFile(name, flag, perm)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	shareRemoveFile = func(share *smb2.Share, name string) error {
		if share == nil {
			return os.ErrInvalid
		}
		return share.Remove(name)
	}
)

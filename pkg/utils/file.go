package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"

	"github.com/pkg/errors"

	"nanto.io/application-auto-scaling-service/pkg/utils/logutil"
)

var logger = logutil.GetLogger()

func CheckFileIsExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		logger.Warningf("CheckFileIsExist os.Stat err: %v", err)
		return !os.IsNotExist(err)
	}
	return true
}

func DataHashMd5(data []byte) string {
	hash := md5.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func FileHashMd5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "open file[%s] err", path)
	}
	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", errors.Wrapf(err, "io copy file[%s] to md5 hash err", path)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

package rotatefile

import (
	"fmt"
	"os"
	"time"

	"github.com/mohanson/doa"
)

// Test whether a path exists. Returns False for broken symbolic links.
func isFileExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// Handler for logging to a set of files, which switches from one file to the next when the current file reaches a
// certain size.
type RotateFile struct {
	Backup   int
	CapLimit int
	CapUsing int
	File     *os.File
	Name     string
	UpdateAt time.Time
}

// Open with flag os.O_WRONLY + os.O_TRUNC
func (f *RotateFile) OpenWronlyTrunca() error {
	r, err := os.OpenFile(f.Name, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	f.File = r
	return nil
}

// Open with flag os.O_WRONLY + os.O_CREATE + os.O_APPEND
func (f *RotateFile) OpenWronlyCreateAppend() error {
	r, err := os.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	f.File = r
	return nil
}

func (f *RotateFile) write(b []byte) (n int, err error) {
	capSpace := f.CapLimit - f.CapUsing
	if capSpace >= len(b) {
		n, err = f.File.Write(b)
		f.CapUsing += n
		f.UpdateAt = time.Now()
		return
	}
	writeN := 0
	writeN, err = f.File.Write(b[:capSpace])
	n += writeN
	f.CapUsing += writeN
	f.UpdateAt = time.Now()
	if err != nil {
		return
	}

	err = f.File.Close()
	if err != nil {
		return
	}

	// Rollover occurs whenever the current log file is nearly maxBytes in length. If backupCount is >= 1, the system
	// will successively create new files with the same pathname as the base file, but with extensions ".1", ".2" etc.
	// appended to it. For example, with a backupCount of 5 and a base file name of "app.log", you would get "app.log",
	// "app.log.1", "app.log.2", ... through to "app.log.5". The file being written to is always "app.log" - when it
	// gets filled up, it is closed and renamed to "app.log.1", and if files "app.log.1", "app.log.2" etc. exist, then
	// they are renamed to "app.log.2", "app.log.3" etc. respectively.
	//
	// If maxBytes is zero, rollover never occurs.
	if f.Backup > 0 {
		for i := f.Backup - 1; i > 0; i-- {
			sfn := fmt.Sprintf("%s.%d", f.Name, i)
			dfn := fmt.Sprintf("%s.%d", f.Name, i+1)
			if isFileExist(sfn) {
				if isFileExist(dfn) {
					err = os.Remove(dfn)
					if err != nil {
						return
					}
				}
				err = os.Rename(sfn, dfn)
				if err != nil {
					return
				}
			}
		}
		dfn := fmt.Sprintf("%s.%d", f.Name, 1)
		if isFileExist(dfn) {
			err = os.Remove(dfn)
			if err != nil {
				return
			}
		}
		err = os.Rename(f.Name, dfn)
		if err != nil {
			return
		}
	}

	err = f.OpenWronlyCreateAppend()
	if err != nil {
		return
	}
	writeN, err = f.File.Write(b[capSpace:])
	n += writeN
	f.CapUsing = writeN
	f.UpdateAt = time.Now()
	return
}

// Panic directly to avoid errors being eaten.
func (f *RotateFile) Write(b []byte) (n int, err error) {
	n = doa.Try2(f.write(b)).(int)
	return n, nil
}

// Close closes the File.
func (f *RotateFile) Close() error {
	return f.File.Close()
}

// Open the specified file and use it as the stream for logging.
//
// By default, the file grows indefinitely. You can specify particular values of maxBytes and backupCount to allow the
// file to rollover at a predetermined size.
func New(name string, backup int, size int) (*RotateFile, error) {
	r := &RotateFile{
		Backup:   backup,
		CapLimit: size,
		Name:     name,
	}
	if err := r.OpenWronlyCreateAppend(); err != nil {
		return r, err
	}
	s, err := r.File.Stat()
	if err != nil {
		return nil, err
	}
	r.CapUsing = int(s.Size())
	r.UpdateAt = s.ModTime()
	return r, nil
}

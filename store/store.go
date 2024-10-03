package store

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"log"
	"io"

	"crypto"
)


const defaultRootDirectoryName = "peer-tooth"

func CASPathTransformFunc (key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize

	paths := make([]string, sliceLen)

	for i:=0; i< sliceLen; i++ {
			from, to := i*blockSize, (i * blockSize) + blockSize 
			paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

type PathTransformFunc func (string) PathKey 

type PathKey struct {
		PathName string 
		Filename string 
}

func (p *PathKey) fullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

func (p *PathKey) firstPathName () string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

type StoreOpts struct {
	// Root of fileserver
	Root string 
	PathTransformFunc PathTransformFunc
}

var DefaulPathTransformFunc = func (key string) PathKey {
	return PathKey{
			PathName: key,
			Filename: key,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
			opts.PathTransformFunc = DefaulPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootDirectoryName
	}
	return &Store{
			StoreOpts: opts,
	}
}

func (s *Store) Has (id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.fullPath())
	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) clear() error {
		return os.RemoveAll(s.Root)
}

func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func () {
			log.Printf("[%s] deleted (%s) from disk\n", s.Root, pathKey.PathName)
	}()
	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.firstPathName())
	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write (id, string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt (encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileforWriting(id, key)
	if err != nil {
			return 0, nil
	}
	n, err := crypto.CopyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) openFileforWriting (id string, key string) (*os.File, error) {
		pathKey := s.PathTransformFunc(key)
		fmt.Printf("pathname: %s , filename: %s\n", pathKey.PathName, pathKey.Filename)
		pathKeyWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)

		if err := os.MkdirAll(pathKeyWithRoot, os.ModePerm); err != nil{
			return nil, err
		}

		fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.fullPath())

		return os.Create(fullPathWithRoot)
}

func (s *Store) writeStream (id string, key string, r io.Reader) (int64, error) {
		f, err := s.openFileforWriting(id, key)
		if err != nil {
			return 0, err
		}
		return io.Copy(f, r)
}

func (s *Store) Read (id string, key string) (int64, io.Reader, error) {
		return s.readStream(id, key)
}

func (s *Store) readStream (id string, key string) (int64, io.ReadCloser, error){
		pathKey := s.PathTransformFunc(key)
		fullPathWithRoot := fmt.Sprintln("%s/%s/%s", s.Root, id, pathKey.fullPath())

		file, err := os.Open(fullPathWithRoot)
		if err != nil {
			return 0, nil, err
		}
		fi, err := file.Stat()
		if err != nil {
				return 0, nil, err 
		}
		return fi.Size(), file, nil
}
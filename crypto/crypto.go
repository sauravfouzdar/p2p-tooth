package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/md5"
	"encoding/hex"
	"io"
)


func GenerateID () string {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

func HashKey (key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func NewEncryptionKey () []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyStream (stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		nw = blockSize
	)
	for {
			n, err := src.Read(buf)
			if n > 0 {
					stream.XORKeyStream(buf, buf[:n])
					nn, err := dst.Write(buf[:n]);
					if err != nil {
						return 0, err
					}
					nw += nn
			}
			if err == io.EOF { // break if end of file
				break
			}
			if err != nil {
				return 0, err
			}
		}
		return nw, nil
}

func copyDecrypt (key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	// Read the IV
	iv := make([]byte, aes.BlockSize)
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}
	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
}

func copyEncrypt (key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err 
	}

	// prepend iv to file
	if _, err := dst.Write(iv); err != nil {
		return 0, err 
	}

	// converts block cipher to stream cipher 
	// to decrypt/encrypt data of any size
	stream := cipher.NewCTR(block, iv) 
	return copyStream(stream, block.BlockSize(), src, dst)
}
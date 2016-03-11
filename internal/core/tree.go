package core

import (
	"io"
	"bytes"
	"errors"
	"fmt"
	"encoding/json"
	"sort"
	"strings"
	"encoding/hex"
	"github.com/s3git/s3git-go/internal/cas"
)

const TREE="tree"

type treeObject struct {
	coreObject
	S3gitAdded   []string `json:"s3gitAdded"`
	S3gitRemoved []string `json:"s3gitRemoved"`
	S3gitPadding string   `json:"s3gitPadding"`
}

func makeTreeObject(added <-chan []byte, removed []string) *treeObject {
	to := treeObject{coreObject: coreObject{S3gitVersion: 1, S3gitType: TREE}}

	addedArray := []string{}

	for k := range added {
		addedArray = append(addedArray, hex.EncodeToString(k))
	}

	sort.Strings(addedArray)
	to.S3gitAdded = addedArray

	sort.Strings(removed)
	to.S3gitRemoved = removed

	return &to
}

func (to *treeObject) empty() bool {
	return len(to.S3gitAdded) == 0 && len(to.S3gitRemoved) == 0
}

func (to *treeObject) writeToDisk() (string, error) {

	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(to); err != nil {
		return "", err
	}

	return to.write(buf, TREE)
}

// Return tree object based on hash
func GetTreeObject(hash string) (*treeObject, error) {

	cr := cas.MakeReader(hash)
	if cr == nil {
		return nil, errors.New(fmt.Sprint("Failed to read hash %s", hash))
	}

	buf := bytes.NewBuffer(nil)
	// TODO: Find out why io.Copy does not read whole file from cas (truncated for 50 MB tree files)
	// io.Copy(buf, cr)

	size := 0
	array := make([]byte, cas.ChunkSize)
	for {
		read, err := cr.Read(array)
		size += read
		if read > 0 {
			_, err := buf.Write(array[:read])
			if err != nil {
				panic(err)
			}
		}
		if err == io.EOF {
			break
		}
	}

	s := string(buf.Bytes())

	dec := json.NewDecoder(strings.NewReader(s))
	var to treeObject
	if err := dec.Decode(&to); err != nil {
		return nil, err
	}

	return &to, nil
}
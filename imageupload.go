package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

func UploadImage(w http.ResponseWriter, req *http.Request) (url string) {
	file, header, err := req.FormFile("file")
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	defer file.Close()

	//log.Printf("\nheader\n %+v\n", header)

	url = urlHost() + "uploads/"

	ext := path.Ext(header.Filename)
	seq := randSeq(20)

	out, err := os.Create("./public/uploads/" + seq + ext)
	if err != nil {
		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
		return
	}

	defer out.Close()

	// write the content from POST to the file
	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Fprintln(w, err)
	}

	url += seq + ext
	log.Println("url: ", url)
	return
}

// https://github.com/noll/mjau/blob/master/util/util.go#L42
// http://stackoverflow.com/a/12527546/24802
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

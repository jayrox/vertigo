package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/martini-contrib/render"
)

func UploadImage(w http.ResponseWriter, req *http.Request, res render.Render) {
	file, header, err := req.FormFile("file")
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	defer file.Close()

	url := urlHost() + "uploads/"

	ext := path.Ext(header.Filename)
	seq := randSeq(20)

	out, err := os.Create("./public/uploads/" + seq + ext)
	if err != nil {
		res.JSON(500, map[string]interface{}{"error": "Unable to create the file for writing. Check your write access privilege"})
		return
	}

	defer out.Close()

	// write the content from POST to the file
	_, err = io.Copy(out, file)
	if err != nil {
		res.JSON(500, map[string]interface{}{"error": err})
		return
	}

	url += seq + ext

	log.Println("url: ", url)
	res.JSON(200, map[string]interface{}{"link": url})
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

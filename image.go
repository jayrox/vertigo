package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

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

func UploadedImages(w http.ResponseWriter, req *http.Request, res render.Render) {
	var fl []string
	files, _ := ioutil.ReadDir("./public/uploads/")
	for _, f := range files {
		//fmt.Println(f.Name())
		fl = append(fl, "/uploads/"+f.Name())
	}
	res.JSON(200, fl)
}

type ImageSrc struct {
	Src string `form:"src"`
}

func DeleteImage(w http.ResponseWriter, req *http.Request, res render.Render, image ImageSrc) {
	log.Println("Deleting: ", image.Src)

	// Validate path
	src_parts := strings.Split(image.Src, "/")
	path := src_parts[1]
	img := src_parts[2]
	log.Println("path: " + path)
	if path != "uploads" {
		log.Println("invalid path: " + path)
		res.JSON(500, map[string]interface{}{"error": "Invalid path."})
		return
	}

	// Validate file extension
	img_parts := strings.Split(img, ".")
	log.Println("ext: " + img_parts[1])
	valid_ext := []string{"png", "jpg", "jpeg", "gif", "webp"}
	if !stringInSlice(img_parts[1], valid_ext) {
		log.Println("invalid ext: " + img_parts[1])
		res.JSON(500, map[string]interface{}{"error": "Invalid file."})
		return
	}

	// Validate file exists
	img_path := "./public/uploads/" + img
	if !Exists(img_path) {
		log.Println("invalid file: " + img_path)
		res.JSON(404, map[string]interface{}{"error": "Invalid file: File not found."})
		return
	}

	// Delete file
	err := os.Remove(img_path)
	if err != nil {
		log.Println("delete failed: ", err)
		res.JSON(500, map[string]interface{}{"error": err})
		return
	}

	log.Println("File: " + img_path + " removed.")
	res.JSON(200, map[string]interface{}{"success": img_path + " has been deleted."})
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

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

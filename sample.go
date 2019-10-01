package main

import (
	"archive/zip"
	"fmt"
	"log"
)

func main() {
	// Open a zip archive for reading.
	r, err := zip.OpenReader("/Users/ilya/Downloads/adsb/2019-06-01.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		/*rc, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.CopyN(os.Stdout, rc, 68)
		if err != nil {
			log.Fatal(err)
		}
		rc.Close()*/
		fmt.Println()
	}
}

package ascii

import (
	"fmt"
	"io/ioutil"
)

func DisplayArt(filename string) {
	art, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Remote Activity Moderation Tool")
		return
	}
	fmt.Print(string(art))
	fmt.Println()
}

func DisplayDefaultArt() {
	DisplayArt("ascii/art.txt")
}

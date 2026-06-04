package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	path := "/home/salman/Projects/research/Sorush/soroush-relay/researchs/web.splus.ir/resources/js/7532.8a7998b226a70b0f978d.js"
	data, err := os.ReadFile(path)
	if err != nil {
		path = "/home/salman/Projects/research/Sorush/soroush-relay/researchs/web.splus.ir/resources/js/main.e0f8f960d196367e0c8b.js"
		data, err = os.ReadFile(path)
		if err != nil {
			panic(err)
		}
	}

	content := string(data)
	// We want to find all update constructors like: updateNewMessage#1f240ced ... = Update;
	// Soroush TL schema definitions in JS are usually formatted as standard TL schema strings inside quotes.
	// Let's find all schemas starting with a letter and containing '#' and ending with ';' or similar.
	// Let's use a regex to find all constructors of the form word#hex fields = type;
	re := regexp.MustCompile(`([A-Za-z][A-Za-z0-9\._]*#[0-9a-fA-F]+[^;]*;)`)
	matches := re.FindAllString(content, -1)
	
	fmt.Printf("Found %d constructor strings\n", len(matches))
	for _, m := range matches {
		clean := strings.ReplaceAll(m, "\\n", "\n")
		clean = strings.ReplaceAll(clean, "\\t", "\t")
		lines := strings.Split(clean, "\n")
		for _, line := range lines {
			if strings.Contains(line, "#") && (strings.Contains(line, "= Update;") || strings.Contains(line, "= Updates;")) {
				fmt.Println(strings.TrimSpace(line))
			}
		}
	}
}

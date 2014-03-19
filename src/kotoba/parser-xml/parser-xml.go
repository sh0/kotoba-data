//
// Kotoba
// Copyright (C) 2013 sh0 <sh0@yutani.ee>
//

// Package and imports
package main
import (
    // System
    "flag"
    "fmt"
    "os"
    "bufio"
)

// Main
func main() {
    // Flags
    fn := flag.String("data", "", "Data .xml file")
    num := flag.Int("lines", 0, "Number of lines in data")
    flag.Parse()
    if (*fn == "") {
        fmt.Print("Please specify data file!\n")
        return
    } else if (*num <= 0) {
        fmt.Print("Please specify number of lines in data file!\n")
        return
    }

    // File
    fs, err := os.Open(*fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return
    }
    
    // Header
    fmt.Print("<?xml version=\"1.1\" encoding=\"UTF-8\" ?>\n")
    fmt.Printf("<data entries=\"%d\">\n", *num)
    
    // Line reader
    reader := bufio.NewReader(fs)
    err = nil
    for err == nil {
        // Read line
        var line string
        line, err = reader.ReadString('\n')
        if (err != nil) { break }
        
        fmt.Print(line)
    }
    
    // Footer
    fmt.Print("</data>\n")
}

//
// Kotoba
// Copyright (C) 2013 sh0 <sh0@yutani.ee>
//

// Package and imports
package main
import (
    // System
    "fmt"
    "os"
    "bufio"
    "encoding/xml"
    "encoding/hex"
    "bytes"
    "encoding/binary"
    "strings"
    "strconv"
)

var g_bo binary.ByteOrder

// Dictionary structures
type DictRoot struct {
    XMLName xml.Name `xml:"JMdict"`
    Entry []*DictEntry `xml:"entry"`
}

type DictEntry struct {
    Id string `xml:"ent_seq"`
    Kele []DictKele `xml:"k_ele"`
    Rele []DictRele `xml:"r_ele"`
    Sense []DictSense `xml:"sense"`
    Jlpt int
}

type DictKele struct {
    Keb string `xml:"keb"`
    KePri []string `xml:"ke_pri"`
}

type DictRele struct {
    Reb string `xml:"reb"`
    RePri []string `xml:"re_pri"`
}

type DictSense struct {
    Pos string `xml:"pos"`
    Gloss []string `xml:"gloss"`
}

// Save structure
type WordSaveRoot struct {
    XMLName xml.Name `xml:"Words"`
    Entry []WordSaveEntry
}

type WordSaveEntry struct {
    Id string
    Kele []string
    Rele []string
    Sense []WordSaveSense
    Cat []string
}

type WordSaveSense struct {
    Pos string
    Gloss []string
}

// Load tanos JLPT levels
type WordTanos struct {
    Hash string
    Jlpt int
    Used bool
}

func LoadTanos() (map[string]*WordTanos, map[string]*WordTanos) {
    // File
    fs, err := os.Open("words-tanos.pipe")
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return nil, nil
    }
    
    // Kanji and kana maps
    mk := map[string]*WordTanos{}
    mr := map[string]*WordTanos{}
    
    // Line reader
    reader := bufio.NewReader(fs)
    err = nil
    for err == nil {
        // Read line
        var line string
        line, err = reader.ReadString('\n')
        if (err != nil) { break }
        
        // Split
        record := strings.Split(line, "\t")
        if (len(record) < 5) {
            fmt.Printf("Error: Line does not have enough columns! num=%d\n", len(record))
            continue
        }
        
        // Hash
        hash := strings.TrimSpace(record[0])
        
        // Get kanji and kana
        has_real := false
        jp_real := strings.Split(strings.TrimSpace(record[1]), ";")
        for i := range jp_real {
            jp_real[i] = strings.TrimSpace(jp_real[i])
            if len(jp_real[i]) > 0 { has_real = true }
        }
        has_kana := false
        jp_kana := strings.Split(strings.TrimSpace(record[2]), ";")
        for i := range jp_kana {
            jp_kana[i] = strings.TrimSpace(jp_kana[i])
            if len(jp_kana[i]) > 0 { has_kana = true }
        }
        
        // Level
        jlpt, _ := strconv.Atoi(strings.TrimSpace(record[5]))
        
        // Add to maps
        word := &WordTanos{
            Hash: hash,
            Jlpt: jlpt,
            Used: false,
        }
        if has_kana && has_real {
            for _, str := range jp_real {
                mk[str] = word
            }
            for _, str := range jp_kana {
                mr[str] = word
            }
        } else if has_real {
            for _, str := range jp_real {
                mr[str] = word
            }
        }
    }
    
    // Return
    return mk, mr
}

// Main
func main() {
    // Dictionary
    dict := DictRoot{}
    
    // Tanos wordlist
    jlpt_mk, jlpt_mr := LoadTanos()
    migmap := map[string]string{}
    
    // Save list
    save := WordSaveRoot{
        Entry: []WordSaveEntry{},
    }

    // File
    fs, err := os.Open("jmdicte")
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return
    }
    
    // XML reader
    fmt.Printf("Reading...\n")
    reader := bufio.NewReader(fs)
    decoder := xml.NewDecoder(reader)
    decoder.Strict = false
    err = decoder.Decode(&dict)
    if err != nil {
        fmt.Printf("XML error: %s\n", err.Error())
        return
    }
    for _, entry := range dict.Entry { entry.Jlpt = 0 }
    
    // JLPT matching
    fmt.Printf("JLPT matching...\n")
    for _, entry := range dict.Entry {
        if entry.Jlpt == 0 {
            for _, kele := range entry.Kele {
                word, exists := jlpt_mk[kele.Keb]
                if exists && !word.Used {
                    word.Used = true
                    entry.Jlpt = word.Jlpt
                    migmap[word.Hash] = entry.Id
                    break
                }
            }
        }
    }
    for _, entry := range dict.Entry {
        if entry.Jlpt == 0 {
            for _, rele := range entry.Rele {
                word, exists := jlpt_mr[rele.Reb]
                if exists && !word.Used {
                    word.Used = true
                    entry.Jlpt = word.Jlpt
                    migmap[word.Hash] = entry.Id
                    break
                }
            }
        }
    }
    
    // Reformat
    fmt.Printf("Reformatting...\n")
    for _, entry := range dict.Entry {
        
        // Find categories
        cat := []string{}
        if entry.Jlpt > 0 { cat = append(cat, "n" + strconv.Itoa(entry.Jlpt)) }
        for _, kele := range entry.Kele {
            for _, kepri := range kele.KePri {
                found := false
                for _, c := range cat {
                    if c == kepri { found = true }
                }
                if found == false { cat = append(cat, kepri) }
            }
        }
        for _, rele := range entry.Rele {
            for _, repri := range rele.RePri {
                found := false
                for _, c := range cat {
                    if c == repri { found = true }
                }
                if found == false { cat = append(cat, repri) }
            }
        }
        
        // Poulate save entry
        sentry := WordSaveEntry{
            Id: entry.Id,
            Kele: []string{},
            Rele: []string{},
            Sense: []WordSaveSense{},
            Cat: cat,
        }
        for _, kele := range entry.Kele {
            sentry.Kele = append(sentry.Kele, kele.Keb)
        }
        for _, rele := range entry.Rele {
            sentry.Rele = append(sentry.Rele, rele.Reb)
        }
        for _, sense := range entry.Sense {
            pos := strings.Replace(sense.Pos, "&", "", -1)
            pos = strings.Trim(pos, ";")
            ssense := WordSaveSense{
                Pos: pos,
                Gloss: sense.Gloss,
            }
            sentry.Sense = append(sentry.Sense, ssense)
        }
        
        // Add entry
        save.Entry = append(save.Entry, sentry)
        
        /*
        str := ""
        for _, kele := range entry.Kele {
            if len(str) != 0 { str += ";" }
            str += kele.Keb
        }
        fmt.Printf("id='%s', kele='%s'\n", entry.EntSeq, str)
        */
    }
    
    // Save xml
    WriteWords(&save)
    
    // Migration map
    fmt.Printf("Writing migration map...\n")
    WriteMigmap(migmap)
}

func WriteWords(save *WordSaveRoot) {
    // Marshal
    data, err := xml.Marshal(save)
    if err != nil {
        fmt.Printf("XML marshalling error: %s\n", err.Error())
        return
    }
    
    // File
    fs, err := os.OpenFile("out-words.xml", os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if (err != nil) {
        fmt.Printf("Failed to open words xml output file: %s\n", err.Error())
        return
    }
    wr := bufio.NewWriter(fs)
    
    // Write
    wr.Write(data)
    
    // Close
    wr.Flush()
    fs.Close()
}

func WriteMigmap(migmap map[string]string) {
    // Byte order
    g_bo = binary.LittleEndian
    
    // File
    fs, err := os.OpenFile("out-migmap.kdb", os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if (err != nil) {
        fmt.Printf("Failed to open hash migration output file: %s\n", err.Error())
        return
    }
    wr := bufio.NewWriter(fs)
    
    // Write size
    buf_sz := bytes.NewBuffer(nil)
    binary.Write(buf_sz, g_bo, uint32(len(migmap)))
    wr.Write(buf_sz.Bytes())
    
    // Write lines
    for hash, id := range migmap {
        buf := bytes.NewBuffer(nil)
        
        //fmt.Printf("%s -> %s\n", hash, id)
        hash_b, _ := hex.DecodeString(hash)
        id_b, _ := strconv.Atoi(id)
        buf.Write(hash_b)
        binary.Write(buf, g_bo, uint32(id_b))
        
        wr.Write(buf.Bytes())
    }

    // Close
    wr.Flush()
    fs.Close()
}

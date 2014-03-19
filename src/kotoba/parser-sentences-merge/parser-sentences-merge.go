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
    "strings"
    "strconv"
    "unicode/utf8"
    "math/rand"
    "encoding/hex"
    "sort"
)

// <===> XML <=================================================================>
type XmlOutput struct {
    fs *os.File
    wr *bufio.Writer
}

func XmlOpen(fn string) *XmlOutput {
    // File
    this := &XmlOutput{}
    var err error
    this.fs, err = os.OpenFile(fn, os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if (err != nil) {
        fmt.Printf("Failed to open xml output file: %s\n", err.Error())
        return nil
    }
    this.wr = bufio.NewWriter(this.fs)
    
    // Success
    return this
}

func (this *XmlOutput) Write(str string) {
    this.wr.WriteString(str)
}

func (this *XmlOutput) Close() {
    this.wr.Flush()
    this.fs.Close()
}

func (this *XmlOutput) Escape(str string) string {
    str = strings.Replace(str, "&", "&amp;", -1)
    str = strings.Replace(str, "\"", "&quot;", -1)
    str = strings.Replace(str, "'", "&apos;", -1)
    str = strings.Replace(str, "<", "&lt;", -1)
    str = strings.Replace(str, ">", "&gt;", -1)
    return str
}

// <===> Categories <==========================================================>
type CategoryInfo struct {
    // Info
    Id int
    Name string
    Folder string
}

type CategoryClass struct {
    Info []*CategoryInfo
    Jlpt map[int]*CategoryInfo
}

func CategoryNew() *CategoryClass {
    // Instance
    this := &CategoryClass{
        Info: []*CategoryInfo{},
        Jlpt: map[int]*CategoryInfo{},
    }
    
    // Success
    return this
}

func (this *CategoryClass) AssignId() {
    id := 1
    for _, info := range this.Info {
        info.Id = id
        id++
    }
}

func (this *CategoryClass) LoadJlpt() bool {
    // Default JLPT levels
    for i := 5; i > 0; i-- {
        // Category
        info := &CategoryInfo{
            Id: -1,
            Name: "JLPT N" + strconv.Itoa(i),
            Folder: "Japanese-Language Proficiency Test",
        }
        
        // Insert
        this.Info = append(this.Info, info)
        this.Jlpt[i] = info
    }
    
    // Success
    return true
}

func (this *CategoryClass) Save(fn string) {
    // File
    xml := XmlOpen(fn)
    if (xml == nil) { return }
    
    // Write
    for _, info := range this.Info {
        if (info.Id < 0) { continue }
        str := fmt.Sprintf("<category id=\"%d\" name=\"%s\" folder=\"%s\" />\n", info.Id, info.Name, info.Folder)
        xml.Write(str)
    }
    
    // Close
    xml.Close()
}

// <===> Words <===============================================================>
type WordInfo struct {
    // Info
    Id int
    Hash string
    JpReal string
    JpKana string
    En string
    Flags string
    Level string
    // Meta
    Sref []WordSref
    Cref []*CategoryInfo
}

func (this* WordInfo) LimitSref(num int) {
    // Randomly remove entries until under limit
    for len(this.Sref) > num {
        n := rand.Intn(len(this.Sref))
        narr := make([]WordSref, len(this.Sref) - 1)
        copy(narr, this.Sref[0:n])
        copy(narr[n:], this.Sref[n + 1:])
        this.Sref = narr
    }
}

type WordSref struct {
    Info *SentenceInfo
    Start int
    End int
}

type WordClass struct {
    Info WordInfoSort
}

type WordInfoSort []*WordInfo
func (list WordInfoSort) Len() int { return len(list) }
func (list WordInfoSort) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list WordInfoSort) Less(i, j int) bool {
    a, _ := hex.DecodeString(list[i].Hash)
    b, _ := hex.DecodeString(list[j].Hash)
    if a == nil || b == nil { return true }
    sz := len(a)
    if len(b) < sz { sz = len(b) }
    for k := 0; k < sz; k += 1 {
        if a[k] < b[k] {
            return true
        } else if a[k] > b[k] {
            return false
        }
    }
    return true
}

func WordNew() *WordClass {
    // Instance
    this := &WordClass{
        Info: []*WordInfo{},
    }
    
    // Success
    return this
}

func (this *WordClass) AssignId() {
    // Sort
    sort.Sort(this.Info)
    
    // Assign ids
    id := 1
    for _, info := range this.Info {
        info.Id = id
        id++
    }
}

func (this *WordClass) Load(fn string) bool {
    // File
    fs, err := os.Open(fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return false
    }
    
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
        
        // Category
        jlpt_level, _ := strconv.Atoi(strings.TrimSpace(record[5]))
        cref := g_category.Jlpt[jlpt_level]
        
        // Word
        this.Info = append(this.Info, &WordInfo{
            // Info
            Id: -1,
            Hash: strings.TrimSpace(record[0]),
            JpReal: strings.TrimSpace(record[1]),
            JpKana: strings.TrimSpace(record[2]),
            En: strings.TrimSpace(record[3]),
            Flags: strings.TrimSpace(record[4]),
            // Meta
            Sref: []WordSref{},
            Cref: []*CategoryInfo{ cref },
        })
    }
    
    // Success
    return true
}

func (this *WordClass) Save(fn string) {
    // File
    xml := XmlOpen(fn)
    if (xml == nil) { return }
    
    // Write
    for _, info := range this.Info {
        // Check
        if (info.Id < 0) { continue }
        
        // Word
        str := fmt.Sprintf("<word_data id=\"%d\" hash=\"%s\" text_jp_k=\"%s\" text_jp_f=\"%s\" text_en=\"%s\" flags=\"%s\" />\n",
            info.Id, info.Hash, xml.Escape(info.JpReal), xml.Escape(info.JpKana), xml.Escape(info.En), info.Flags)
        xml.Write(str)
        
        // Srefs
        for _, sref := range info.Sref {
            str = fmt.Sprintf("<word_sref id=\"%d\" sentence=\"%d\" mark_s=\"%d\" mark_e=\"%d\" />\n",
                info.Id, sref.Info.Id, sref.Start, sref.End)
            xml.Write(str)
        }
        
        // Crefs
        for _, cref := range info.Cref {
            str = fmt.Sprintf("<word_cref id=\"%d\" category=\"%d\" />\n", info.Id, cref.Id)
            xml.Write(str)
        }
    }
    
    // Close
    xml.Close()
}

// <===> Sentences <===========================================================>
type SentenceInfo struct {
    // Info
    Id int
    JpReal string
    JpKana string
    JpBase string
    En string
    // Meta
    Usage int
}

type SentenceClass struct {
    Info []*SentenceInfo
    Base map[string][]WordSref
}

func SentenceNew() *SentenceClass {
    // Instance
    this := &SentenceClass{
        Info: []*SentenceInfo{},
        Base: map[string][]WordSref{},
    }
    
    // Success
    return this
}

func (this *SentenceClass) AssignId() {
    id := 1
    for _, info := range this.Info {
        //if (info.Usage > 0) {
            info.Id = id
            id++
        //}
    }
}

func (this *SentenceClass) Load(fn string) bool {
    // File
    fs, err := os.Open(fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return false
    }
    
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
        if (len(record) < 3) {
            fmt.Printf("Error: Line does not have enough columns! num=%d\n", len(record))
            continue
        }
        
        // Info
        info := &SentenceInfo{
            // Info
            Id: -1,
            JpReal: strings.TrimSpace(record[0]),
            JpKana: strings.TrimSpace(record[1]),
            JpBase: strings.TrimSpace(record[3]),
            En: strings.TrimSpace(record[2]),
            // Meta
            Usage: 0,
        }
        this.Info = append(this.Info, info)
        
        // Base
        blist := strings.Split(strings.TrimSpace(record[3]), ";")
        for _, bitem := range blist {
            // Check validity
            dlist := strings.Split(bitem, "@")
            if (len(dlist) < 4) { continue }
            
            // Format mark for base
            start, _ := strconv.Atoi(strings.TrimSpace(dlist[2]))
            end, _ := strconv.Atoi(strings.TrimSpace(dlist[3]))
            
            // Base
            base := WordSref{
                Info: info,
                Start: start,
                End: end,
            }
            
            // Insert
            _, bvalid := this.Base[dlist[0]]
            if (bvalid) {
                this.Base[dlist[0]] = append(this.Base[dlist[0]], base)
            } else {
                this.Base[dlist[0]] = []WordSref{ base }
            }
        }
    }
    
    // Success
    return true
}

func (this *SentenceClass) Save(fn string) {
    // File
    xml := XmlOpen(fn)
    if (xml == nil) { return }
    
    // Write
    for _, info := range this.Info {
        if (info.Id < 0) { continue }
        str := fmt.Sprintf("<sentence id=\"%d\" text_jp=\"%s\" text_en=\"%s\" text_r=\"%s\" text_b=\"%s\" />\n",
            info.Id, xml.Escape(info.JpKana), xml.Escape(info.En), xml.Escape(info.JpReal), xml.Escape(info.JpBase))
        xml.Write(str)
    }
    
    // Close
    xml.Close()
}

func (this *SentenceClass) SearchBase(str string) []WordSref {
    list, found := this.Base[str]
    if (found) {
        return list
    } else {
        return []WordSref{}
    }
}

func (this *SentenceClass) SearchFull(str string) []WordSref {
    list := []WordSref{}
    mark_size := mark_rune_len(str)
    for _, info := range this.Info {
        index := strings.Index(info.JpReal, str)
        if (index >= 0) {
            mark_start := mark_rune_len(info.JpReal[0:index])
            sref := WordSref{
                Info: info,
                Start: mark_start,
                End: mark_start + mark_size,
            }
            list = append(list, sref)
        }
    }
    return list
}

// <===> Utility <=============================================================>
func mark_rune_len(str string) int {
    total := 0
    for (len(str) > 0) {
        _, sz := utf8.DecodeRuneInString(str)
        if (sz == 0) { break }
        total++
        str = str[sz:]
    }
    return total
}

// <===> Main <================================================================>
// Globals
var g_category *CategoryClass
var g_word *WordClass
var g_sentence *SentenceClass

/*
func sref_append(arr *[]WordSref, sref WordSref) []WordSref {
    for _, item := range arr {
        if item.Info == sref.Info { return arr }
    }
    return append(arr, sref)
}
*/
func sref_isdupe(arr []WordSref, sref *WordSref) bool {
    for _, item := range arr {
        if item.Info == sref.Info { return true }
    }
    return false
}

// Main function
func main() {
    // Flags
    fn_w := flag.String("words", "", "Words file")
    fn_s := flag.String("sentences", "", "Sentences file")
    flag.Parse()
    if (*fn_w == "" || *fn_s == "") {
        fmt.Printf("Please specify both words and sentences files!\n")
        return
    }
    
    // Classes
    g_category = CategoryNew()
    g_word = WordNew()
    g_sentence = SentenceNew()

    // Load
    fmt.Print("Loading...\n")
    if (!g_category.LoadJlpt()) {
        fmt.Printf("Error parsing categories!\n")
        return
    }
    if (!g_word.Load(*fn_w)) {
        fmt.Printf("Error parsing words file!\n")
        return
    }
    if (!g_sentence.Load(*fn_s)) {
        fmt.Printf("Error reading sentences file!\n")
        return
    }
    
    // Match
    fmt.Print("Processing... ")
    for num, info := range g_word.Info {
        // Progress meter
        if (num % 100 == 0) {
            fmt.Printf("%.01f%% ", 100.0 * float64(num) / float64(len(g_word.Info)))
        }
        
        // Base word lookup with kanji
        for _, str := range strings.Split(info.JpReal, ";") {
            for _, elem := range g_sentence.SearchBase(str) {
                if !sref_isdupe(info.Sref[0:], &elem) { info.Sref = append(info.Sref, elem) }
            }
        }
        
        // Full text search with kanji
        if (len(info.Sref) < 5) {
            for _, str := range strings.Split(info.JpReal, ";") {
                for _, elem := range g_sentence.SearchFull(str) {
                    if !sref_isdupe(info.Sref[0:], &elem) { info.Sref = append(info.Sref, elem) }
                }
            }
        }

        // Hiragana
        if (len(info.JpKana) > 0 && len(info.Sref) < 2) {
            // Base word lookup with kanji
            for _, str := range strings.Split(info.JpKana, ";") {
                for _, elem := range g_sentence.SearchBase(str) {
                    if !sref_isdupe(info.Sref[0:], &elem) { info.Sref = append(info.Sref, elem) }
                }
            }
            
            // Full text search with kanji
            if (len(info.Sref) == 0) {
                for _, str := range strings.Split(info.JpKana, ";") {
                    for _, elem := range g_sentence.SearchFull(str) {
                        if !sref_isdupe(info.Sref[0:], &elem) { info.Sref = append(info.Sref, elem) }
                    }
                }
            }
        }
            
        // Word sentence references
        info.LimitSref(50)
        for _, sref := range info.Sref {
            sref.Info.Usage++
        }
    }
    fmt.Print("\n")
    
    // Rescan low-hit words
    /*
    for _, info := range g_word.Info {
        if (len(info.Sref) < 4) {
            fmt.Printf("<===> %s (%s) <=======================================>\n", info.JpReal, info.JpKana)
            fmt.Printf("* %s\n", info.En)
            
            // Base word lookup with kanji
            for _, str := range strings.Split(info.JpReal, ";") {
                fmt.Printf("[BK] %s\n", str)
                for i, elem := range g_sentence.SearchBase(str) {
                    fmt.Printf("  * %s\n", elem.Info.JpReal)
                    fmt.Printf("    %s\n", elem.Info.En)
                    if i > 10 {
                        fmt.Printf("  ...\n")
                        break
                    }
                }
            }
            
            // Full text search with kanji
            for _, str := range strings.Split(info.JpReal, ";") {
                fmt.Printf("[FK] %s\n", str)
                for i, elem := range g_sentence.SearchFull(str) {
                    fmt.Printf("  * %s\n", elem.Info.JpReal)
                    fmt.Printf("    %s\n", elem.Info.En)
                    if i > 10 {
                        fmt.Printf("  ...\n")
                        break
                    }
                }
            }
            
            // Hiragana
            if (len(info.JpKana) > 0) {
                // Base word lookup with kanji
                for _, str := range strings.Split(info.JpKana, ";") {
                    fmt.Printf("[BF] %s\n", str)
                    for i, elem := range g_sentence.SearchBase(str) {
                        fmt.Printf("  * %s\n", elem.Info.JpReal)
                        fmt.Printf("    %s\n", elem.Info.En)
                        if i > 10 {
                            fmt.Printf("  ...\n")
                            break
                        }
                    }
                }
                
                // Full text search with kanji
                for _, str := range strings.Split(info.JpKana, ";") {
                    fmt.Printf("[FF] %s\n", str)
                    for i, elem := range g_sentence.SearchFull(str) {
                        fmt.Printf("  * %s\n", elem.Info.JpReal)
                        fmt.Printf("    %s\n", elem.Info.En)
                        if i > 10 {
                            fmt.Printf("  ...\n")
                            break
                        }
                    }
                }
            }
        }
    }
    */
    
    // Id generation
    fmt.Printf("Assigning ids...\n")
    g_category.AssignId()
    g_word.AssignId()
    g_sentence.AssignId()
    
    // Save
    fmt.Print("Writing xml...\n")
    g_category.Save("out-category.xml")
    g_word.Save("out-word.xml")
    g_sentence.Save("out-sentence.xml")
    
    // Word debug
    /*
    for _, word := range data_w {
        if len(word.Ref) > 3 { continue }
        fmt.Print("<=========================================================================>\n")
        fmt.Printf("%s - %s - %s\n", word.JpReal, word.JpKana, word.En)
        for _, ref := range word.Ref {
            fmt.Printf("* [%d, %d] %s\n", ref.Start, ref.End, ref.Obj.JpKana)
            fmt.Printf("  %s\n", ref.Obj.En)
        }
    }
    */
    
    // Sentence statistics
    /*
    fmt.Print("<=========================================================================>\n")
    fmt.Printf("Sentence statistics:\n")
    stats_s_lot := 0
    stats_s_arr := make([]int, 200)
    for i := range stats_s_arr { stats_s_arr[i] = 0 }
    for _, entry := range data_s {
        if (entry.Count < len(stats_s_arr)) {
            stats_s_arr[entry.Count]++
        } else {
            stats_s_lot++
        }
    }
    for i, n := range stats_s_arr {
        fmt.Printf("* %3d => %5d\n", i, n)
    }
    fmt.Printf("* ... => %5d\n", stats_s_lot)
    */
    
    // Word statistics
    fmt.Print("<=========================================================================>\n")
    fmt.Printf("Word statistics:\n")
    stats_w_lot := 0
    stats_w_arr := make([]int, 10)
    for i := range stats_w_arr { stats_w_arr[i] = 0 }
    for _, info := range g_word.Info {
        /*
        if (len(info.Sref) == 0) {
            fmt.Printf("%s\t%s\t%s\n", info.JpReal, info.JpKana, info.En)
        }
        */
        if (len(info.Sref) < len(stats_w_arr)) {
            stats_w_arr[len(info.Sref)]++
        } else {
            stats_w_lot++
        }
    }
    for i, n := range stats_w_arr {
        fmt.Printf("* %3d => %5d\n", i, n)
    }
    fmt.Printf("* ... => %5d\n", stats_w_lot)
}

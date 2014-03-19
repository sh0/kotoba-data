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
    "io"
    "bufio"
    "bytes"
    "strings"
    "strconv"
    "crypto/sha1"
    "unicode"
    "unicode/utf8"
    "encoding/binary"
    "encoding/xml"
    "sort"
    "math/rand"
    "runtime"
)

var g_bo binary.ByteOrder

func hash_sha1(str string) []byte {
    sha := sha1.New()
    io.WriteString(sha, str)
    return sha.Sum(nil)
}

// <===> XML <=================================================================>
type DataOutput struct {
    fs *os.File
    wr *bufio.Writer
}

func DataOpen(fn string) *DataOutput {
    // File
    this := &DataOutput{}
    var err error
    this.fs, err = os.OpenFile(fn, os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if (err != nil) {
        fmt.Printf("Failed to open data output file: %s\n", err.Error())
        return nil
    }
    this.wr = bufio.NewWriter(this.fs)
    
    // Success
    return this
}

func (this *DataOutput) Write(data []byte) {
    this.wr.Write(data)
}

func (this *DataOutput) Close() {
    this.wr.Flush()
    this.fs.Close()
}

// <===> Categories <==========================================================>
type CategoryInfo struct {
    // Info
    Name string
    Words WordInfoIdent
    // Marshal
    Id int
    Offset int
    Marshal []byte
}

type CategoryInfoName []*CategoryInfo
func (list CategoryInfoName) Len() int { return len(list) }
func (list CategoryInfoName) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list CategoryInfoName) Less(i, j int) bool { return list[i].Name < list[j].Name }

type CategoryClass struct {
    // Info
    Info CategoryInfoName
    Label map[string]*CategoryInfo
    // Data
    Data []byte
}

func CategoryNew() *CategoryClass {
    // Instance
    this := &CategoryClass{
        // Info
        Info: []*CategoryInfo{},
        Label: map[string]*CategoryInfo{},
    }
    
    // Success
    return this
}

func (this *CategoryClass) AssignId() {
    // Sort list
    sort.Sort(this.Info)
    
    // Assign ids
    id := 0
    for i := range this.Info {
        this.Info[i].Id = id
        id += 1
    }
}

func (this *CategoryClass) Marshal() {
    // Marshal data
    offset := 0
    for _, info := range this.Info {
        // Reorder word list
        sort.Sort(info.Words)
    
        // Data
        info.Offset = offset
        buf := bytes.NewBuffer(nil)
        
        binary.Write(buf, g_bo, uint16(len(info.Name)))
        buf.WriteString(info.Name)
        
        binary.Write(buf, g_bo, uint32(len(info.Words)))
        for _, word := range info.Words {
            binary.Write(buf, g_bo, uint32(word.Id))
        }
        
        info.Marshal = buf.Bytes()
        offset += len(info.Marshal)
    }
    
    // Data buffer
    buf := bytes.NewBuffer(nil)
    
    // Number of entries
    binary.Write(buf, g_bo, uint32(len(this.Info)))
    
    // Index
    for _, info := range this.Info {
        binary.Write(buf, g_bo, uint32(info.Offset))
    }
    binary.Write(buf, g_bo, uint32(offset))
    
    // Entries
    for _, info := range this.Info {
        buf.Write(info.Marshal)
    }
    
    // Data result
    this.Data = buf.Bytes()
}

func (this *CategoryClass) Load() bool {
    // Default JLPT levels
    for i := 5; i > 0; i-- {
        // Category
        info := &CategoryInfo{
            Name: "Japanese-Language Proficiency Test/JLPT N" + strconv.Itoa(i),
            Words: []*WordInfo{},
        }
        
        // Insert
        this.Info = append(this.Info, info)
        this.Label["n" + strconv.Itoa(i)] = info
    }
    
    // Mainichi Shinbun most popular
    info_news1 := &CategoryInfo{
        Name: "Mainichi Shimbun newspaper/Common words #1",
        Words: []*WordInfo{},
    }
    this.Info = append(this.Info, info_news1)
    this.Label["news1"] = info_news1
    
    info_news2 := &CategoryInfo{
        Name: "Mainichi Shimbun newspaper/Common words #2",
        Words: []*WordInfo{},
    }
    this.Info = append(this.Info, info_news2)
    this.Label["news2"] = info_news2
    
    // Ichimango goi bunruishuu most popular
    info_ichi1 := &CategoryInfo{
        Name: "Ichimango goi bunruishuu/Common words",
        Words: []*WordInfo{},
    }
    this.Info = append(this.Info, info_ichi1)
    this.Label["ichi1"] = info_ichi1
    
    // Frequency of use words
    for i := 0; i < 20; i += 1 {
        name_freq := fmt.Sprintf("Mainichi Shimbun newspaper/Most frequent words #%02d", i + 1)
        info_freq := &CategoryInfo{
            Name: name_freq,
            Words: []*WordInfo{},
        }
        this.Info = append(this.Info, info_freq)
        this.Label[fmt.Sprintf("nf%02d", i + 1)] = info_freq
    }
    
    // Success
    return true
}

func (this *CategoryClass) Save(fn string) {
    fs := DataOpen(fn)
    if (fs == nil) { return }
    fs.Write(this.Data)
    fs.Close()
}

// <===> Words <===============================================================>
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

// Info structure
type WordInfo struct {
    // Info
    Ident int
    Kele []string
    Rele []string
    Sense []WordSaveSense
    Cref []*CategoryInfo
    Sref []*SentenceBref
    // Marshal
    Id int
    Offset int
    Marshal []byte
}

type WordInfoIdent []*WordInfo
func (list WordInfoIdent) Len() int { return len(list) }
func (list WordInfoIdent) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list WordInfoIdent) Less(i, j int) bool { return list[i].Ident < list[j].Ident }

type WordRank struct {
    Info *WordInfo
    Rank int
}

type WordClass struct {
    // Info
    Info WordInfoIdent
    BaseReal map[string][]WordRank
    BaseKana map[string][]WordRank
    BaseEn map[string][]WordRank
    // Data
    Data []byte
}

func WordNew() *WordClass {
    // Instance
    this := &WordClass{
        // Info
        Info: []*WordInfo{},
        BaseReal: map[string][]WordRank{},
        BaseKana: map[string][]WordRank{},
        BaseEn: map[string][]WordRank{},
    }
    
    // Success
    return this
}

func (this *WordClass) AssignId() {
    // Sort list
    sort.Sort(this.Info)
    
    // Assign ids
    id := 0
    for i := range this.Info {
        this.Info[i].Id = id
        id += 1
    }
}

func (this *WordClass) Marshal() {
    // Remove sentence references until under limit
    for _, info := range this.Info {
        for len(info.Sref) > 200 {
            n := rand.Intn(len(info.Sref))
            narr := make([]*SentenceBref, len(info.Sref) - 1)
            copy(narr, info.Sref[0:n])
            copy(narr[n:], info.Sref[n + 1:])
            info.Sref = narr
        }
    }
    
    // Marshal data
    offset := 0
    for i, info := range this.Info {
        this.Info[i].Offset = offset
        buf := bytes.NewBuffer(nil)
        
        binary.Write(buf, g_bo, uint16(len(info.Kele)))
        for _, str := range info.Kele {
            binary.Write(buf, g_bo, uint16(len(str)))
            buf.WriteString(str)
        }
        
        binary.Write(buf, g_bo, uint16(len(info.Rele)))
        for _, str := range info.Rele {
            binary.Write(buf, g_bo, uint16(len(str)))
            buf.WriteString(str)
        }
        
        binary.Write(buf, g_bo, uint16(len(info.Sense)))
        for _, sense := range info.Sense {
            binary.Write(buf, g_bo, uint16(len(sense.Gloss)))
            for _, str := range sense.Gloss {
                binary.Write(buf, g_bo, uint16(len(str)))
                buf.WriteString(str)
            }
        }
        
        binary.Write(buf, g_bo, uint16(len(info.Cref)))
        for _, cref := range info.Cref {
            binary.Write(buf, g_bo, uint16(cref.Id))
        }
        
        binary.Write(buf, g_bo, uint16(len(info.Sref)))
        for _, sref := range info.Sref {
            binary.Write(buf, g_bo, uint32(sref.Info.Id))
            binary.Write(buf, g_bo, uint16(sref.Start))
            binary.Write(buf, g_bo, uint16(sref.End))
        }
        
        this.Info[i].Marshal = buf.Bytes()
        offset += len(this.Info[i].Marshal)
    }
    
    // Data buffer
    buf := bytes.NewBuffer(nil)
    
    // Number of entries
    binary.Write(buf, g_bo, uint32(len(this.Info)))
    
    // Index
    for _, info := range this.Info {
        binary.Write(buf, g_bo, uint32(info.Offset))
    }
    binary.Write(buf, g_bo, uint32(offset))
    
    // Entries
    for _, info := range this.Info {
        buf.Write(info.Marshal)
    }
    
    // Idents
    for _, info := range this.Info {
        binary.Write(buf, g_bo, uint32(info.Ident))
    }
    
    // Data result
    this.Data = buf.Bytes()
}

func (this *WordClass) Load(fn string) bool {
    // File
    fs, err := os.Open(fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return false
    }
    
    // Save structure
    save := WordSaveRoot{}
    
    // XML reader
    reader := bufio.NewReader(fs)
    decoder := xml.NewDecoder(reader)
    decoder.Strict = false
    err = decoder.Decode(&save)
    if err != nil {
        fmt.Printf("XML error: %s\n", err.Error())
        return false
    }
    
    // Parse entries
    for _, entry := range save.Entry {
        // Categories
        cref := []*CategoryInfo{}
        for _, cat := range entry.Cat {
            _, exists := g_category.Label[cat]
            if exists {
                cref = append(cref, g_category.Label[cat])
            }
        }
    
        // Id
        ident, _ := strconv.Atoi(entry.Id)
        
        // Entry
        info := &WordInfo{
            Ident: ident,
            Kele: entry.Kele,
            Rele: entry.Rele,
            Sense: entry.Sense,
            Cref: cref,
            Sref: []*SentenceBref{},
        }
        this.Info = append(this.Info, info)
        
        // Save category references
        for _, c := range cref {
            c.Words = append(c.Words, info)
        }
        
        // Kanji and kana references
        for _, str := range entry.Kele {
            _, exists := this.BaseReal[str]
            if exists {
                this.BaseReal[str] = append(this.BaseReal[str], WordRank{ Info: info, Rank: 0 })
            } else {
                this.BaseReal[str] = []WordRank{ WordRank{ Info: info, Rank: 0 } }
            }
        }
        for _, str := range entry.Rele {
            _, exists := this.BaseKana[str]
            if exists {
                this.BaseKana[str] = append(this.BaseKana[str], WordRank{ Info: info, Rank: 0 })
            } else {
                this.BaseKana[str] = []WordRank{ WordRank{ Info: info, Rank: 0 } }
            }
        }
        
        // Sense references
        rank := 0
        for _, sense := range entry.Sense {
            for _, str := range sense.Gloss {
                arr := this.EnSanitize(str)
                for _, item := range arr {
                    _, exists := this.BaseEn[item]
                    if exists {
                        this.BaseEn[item] = append(this.BaseEn[item], WordRank{ Info: info, Rank: rank })
                    } else {
                        this.BaseEn[item] = []WordRank{ WordRank{ Info: info, Rank: rank } }
                    }
                }
                rank += 1
            }
        }
    }
    
    // Success
    return true
}

func (this *WordClass) EnSanitize(str string) []string {
    // Break to runes
    erune := []rune{}
    for (len(str) > 0) {
        r, sz := utf8.DecodeRuneInString(str)
        if (sz == 0) { break }
        erune = append(erune, r)
        str = str[sz:]
    }
    
    // Check runes and rebuild string (lower case)
    str = ""
    buf := make([]byte, 16)
    for _, r := range erune {
        if !unicode.IsLetter(r) && r != ' ' {
            break
        }
        r = unicode.ToLower(r)
        n := utf8.EncodeRune(buf, r)
        if (n > 0) {
            str += string(buf[0:n])
        }
    }
    
    // Create list
    ret := []string{}
    for _, item := range strings.Split(str, " ") {
        item = strings.TrimSpace(item)
        if len(item) > 2 { ret = append(ret, item) }
        if len(ret) >= 16 { break }
    }
    
    // Return result
    return ret
}

func (this *WordClass) Save(fn string) {
    fs := DataOpen(fn)
    if (fs == nil) { return }
    fs.Write(this.Data)
    fs.Close()
}


/*
func (this *WordClass) Search() {
    
    num_cpu := 4
    runtime.GOMAXPROCS(num_cpu + 1)
    
    sem := make(chan int, num_cpu)
    
    num_unit := (len(this.Info) + num_cpu - 1) / num_cpu
    
    for i_cpu := 0; i_cpu < num_cpu; i_cpu++ {
        num := num_unit
        if (i_cpu * num_unit) + num > len(this.Info) { num = len(this.Info) - (i_cpu * num_unit) }
        
        go func(offset int, num int) {
            for i := 0; i < num; i++ {
                if i % 1000 == 0 { fmt.Printf("%dk ", i / 1000) }
                this.SearchInfo(this.Info[offset + i])
            }
            sem <- 1
        } (i_cpu * num_unit, num)
    }
    
    for i := 0; i < num_cpu; i++ {
        <-sem
    }
    fmt.Printf("\n")
}
*/

func (this *WordClass) Search() {
    
    num_cpu := 4
    runtime.GOMAXPROCS(num_cpu + 1)
    
    sem := make(chan int, len(this.Info))
    
    for _, info := range this.Info {
        go func(info *WordInfo) {
            this.SearchInfo(info)
            sem <- 1
        } (info)
    }
    
    for i := 0; i < len(this.Info); i++ {
        /*
        name := ""
        for _, kele := range this.Info[i].Kele {
            if len(name) > 0 { name += ", " }
            name += kele
        }
        fmt.Printf("%d: %s\n", i, name)
        */
        //if i % 1000 == 0 { fmt.Printf("%dk ", i / 1000) }
        <-sem
    }
    fmt.Printf("\n")
}

func (this *WordClass) SearchInfo(info *WordInfo) {
    name := ""
    for _, kele := range info.Kele {
        if len(name) > 0 { name += ", " }
        name += kele
    }
    fmt.Printf("%d: %s\n", info.Ident, name)

    // Try kanji match
    for _, kele := range info.Kele {
        list, exists := g_sentence.BaseReal[kele]
        if exists {
            for _, item := range list {
                found := false
                for _, sref := range info.Sref {
                    if sref.Info == item.Info { found = true }
                }
                if !found { info.Sref = append(info.Sref, item) }
            }
        }
    }
    if len(info.Sref) > 3 {
        return
    }
    
    // Try kana match
    for _, rele := range info.Rele {
        list, exists := g_sentence.BaseKana[rele]
        if exists {
            for _, item := range list {
                found := false
                for _, sref := range info.Sref {
                    if sref.Info == item.Info { found = true }
                }
                if !found { info.Sref = append(info.Sref, item) }
            }
        }
    }
    if len(info.Sref) > 3 {
        return
    }
    
    // Try kanji full-text search
    for _, kele := range info.Kele {
        list := g_sentence.Search(kele)
        for _, item := range list {
            found := false
            for _, sref := range info.Sref {
                if sref.Info == item.Info { found = true }
            }
            if !found { info.Sref = append(info.Sref, item) }
        }
    }
    if len(info.Sref) > 3 {
        return
    }
}

// <===> Sentences <===========================================================>
type SentenceInfo struct {
    // Info
    Ident int
    JpReal string
    JpKana string
    En string
    // Marshal
    Id int
    Offset int
    Marshal []byte
}

type SentenceInfoIdent []*SentenceInfo
func (list SentenceInfoIdent) Len() int { return len(list) }
func (list SentenceInfoIdent) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list SentenceInfoIdent) Less(i, j int) bool { return list[i].Ident < list[j].Ident }

type SentenceIndex struct {
    Name string
    Offset int
    Info []*SentenceInfo
}

type SentenceBref struct {
    Info *SentenceInfo
    Start int
    End int
}

type SentenceClass struct {
    // Info
    Info SentenceInfoIdent
    BaseReal map[string][]*SentenceBref
    BaseKana map[string][]*SentenceBref
    // Index
    Index []*SentenceIndex
    // Data
    Data []byte
}

func SentenceNew() *SentenceClass {
    // Instance
    this := &SentenceClass{
        // Info
        Info: []*SentenceInfo{},
        BaseReal: map[string][]*SentenceBref{},
        BaseKana: map[string][]*SentenceBref{},
        // Indices
        Index: []*SentenceIndex{},
    }
    
    // Success
    return this
}

func (this *SentenceClass) Search(text string) []*SentenceBref {
    list := []*SentenceBref{}
    mark_size := mark_rune_len(text)
    for _, info := range this.Info {
        index := strings.Index(info.JpReal, text)
        if index >= 0 {
            mark_start := mark_rune_len(info.JpReal[0:index])
            sref := &SentenceBref{
                Info: info,
                Start: mark_start,
                End: mark_start + mark_size,
            }
            list = append(list, sref)
        }
    }
    return list
}

func (this *SentenceClass) AssignId() {
    // Sort
    sort.Sort(this.Info)
    
    // Assign ids
    id := 0
    for _, info := range this.Info {
        info.Id = id
        id++
    }
}

func (this *SentenceClass) Marshal() {
    // Marshal data
    offset := 0
    for i, info := range this.Info {
        this.Info[i].Offset = offset
        buf := bytes.NewBuffer(nil)

        binary.Write(buf, g_bo, uint16(len(info.JpKana)))
        buf.WriteString(info.JpKana)
        
        binary.Write(buf, g_bo, uint16(len(info.En)))
        buf.WriteString(info.En)
        
        this.Info[i].Marshal = buf.Bytes()
        offset += len(this.Info[i].Marshal)
    }
    
    // Data buffer
    buf := bytes.NewBuffer(nil)
    
    // Number of entries
    binary.Write(buf, g_bo, uint32(len(this.Info)))
    
    // Index
    for _, info := range this.Info {
        binary.Write(buf, g_bo, uint32(info.Offset))
    }
    binary.Write(buf, g_bo, uint32(offset))
    
    // Entries
    for _, info := range this.Info {
        buf.Write(info.Marshal)
    }
    
    // Data
    this.Data = buf.Bytes()
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
        if (len(record) < 5) {
            fmt.Printf("Error: Line does not have enough columns! num=%d\n", len(record))
            continue
        }
        
        // Values
        jp_ident, _ := strconv.Atoi(strings.TrimSpace(record[0]))
        jp_real := strings.TrimSpace(record[1])
        jp_kana := strings.TrimSpace(record[2])
        en := strings.TrimSpace(record[3])
        
        // Info
        info := &SentenceInfo{
            // Info
            Ident: jp_ident,
            JpReal: jp_real,
            JpKana: jp_kana,
            En: en,
        }
        this.Info = append(this.Info, info)
        
        // Base kanji and hiragana
        blist := strings.Split(strings.TrimSpace(record[4]), ";")
        for _, bitem := range blist {
            // Check validity
            dlist := strings.Split(bitem, "@")
            if (len(dlist) < 4) { continue }
            
            // Format mark for base
            start, _ := strconv.Atoi(strings.TrimSpace(dlist[2]))
            end, _ := strconv.Atoi(strings.TrimSpace(dlist[3]))
            
            // Insert base kanji
            _, bvalid := this.BaseReal[dlist[0]]
            if (bvalid) {
                bfound := false
                for _, item := range this.BaseReal[dlist[0]] {
                    if item.Info == info { bfound = true }
                }
                if !bfound {
                    this.BaseReal[dlist[0]] = append(
                        this.BaseReal[dlist[0]],
                        &SentenceBref{
                            Info: info,
                            Start: start,
                            End: end,
                        })
                }
            } else {
                this.BaseReal[dlist[0]] = []*SentenceBref{
                    &SentenceBref{
                        Info: info,
                        Start: start,
                        End: end,
                    }}
            }
            
            // Insert base kana
            _, bvalid = this.BaseKana[dlist[1]]
            if (bvalid) {
                bfound := false
                for _, item := range this.BaseKana[dlist[1]] {
                    if item.Info == info { bfound = true }
                }
                if !bfound {
                    this.BaseKana[dlist[1]] = append(
                        this.BaseKana[dlist[1]],
                        &SentenceBref{
                            Info: info,
                            Start: start,
                            End: end,
                        })
                }
            } else {
                this.BaseKana[dlist[1]] = []*SentenceBref{
                    &SentenceBref{
                        Info: info,
                        Start: start,
                        End: end,
                    }}
            }
        }
    }
    
    // Success
    return true
}

func (this *SentenceClass) Save(fn string) {
    fs := DataOpen(fn)
    if (fs == nil) { return }
    fs.Write(this.Data)
    fs.Close()
}

// <===> Base tables <=========================================================>
type BaseInfo struct {
    // Info
    Name string
    Wref []WordRank
    // Marshal
    Id int
    Offset int
    Marshal []byte
}

type BaseInfoName []*BaseInfo
func (list BaseInfoName) Len() int { return len(list) }
func (list BaseInfoName) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list BaseInfoName) Less(i, j int) bool {
    ra := rune_decode(list[i].Name)
    rb := rune_decode(list[j].Name)
    sz := len(ra)
    if sz > len(rb) { sz = len(rb) }
    for i := 0; i < sz; i++ {
        if ra[i] < rb[i] {
            return true
        } else if ra[i] > rb[i] {
            return false
        }
    }
    return len(ra) < len(rb)
}

type BaseClass struct {
    // Info
    Info BaseInfoName
    // Data
    Data []byte
}

func BaseNew() *BaseClass {
    // Instance
    this := &BaseClass{
        // Info
        Info: []*BaseInfo{},
    }
    
    // Success
    return this
}

func (this *BaseClass) AssignId() {
    // Sort
    sort.Sort(this.Info)
    
    // Assign ids
    id := 0
    for _, info := range this.Info {
        info.Id = id
        id++
    }
}

func (this *BaseClass) Marshal() {
    // Marshal data
    offset := 0
    for i, info := range this.Info {
        this.Info[i].Offset = offset
        buf := bytes.NewBuffer(nil)

        binary.Write(buf, g_bo, uint16(len(info.Name)))
        buf.WriteString(info.Name)
        
        binary.Write(buf, g_bo, uint16(len(info.Wref)))
        for _, wref := range info.Wref {
            var rank uint32
            rank = uint32(wref.Rank)
            if (rank < 0) { rank = 0 }
            if (rank > 15) { rank = 15 }
            rank = rank << 28
            binary.Write(buf, g_bo, uint32(uint32(wref.Info.Id) | rank))
        }
        
        this.Info[i].Marshal = buf.Bytes()
        offset += len(this.Info[i].Marshal)
    }
    
    // Data buffer
    buf := bytes.NewBuffer(nil)
    
    // Number of entries
    binary.Write(buf, g_bo, uint32(len(this.Info)))
    
    // Index
    for _, info := range this.Info {
        binary.Write(buf, g_bo, uint32(info.Offset))
    }
    binary.Write(buf, g_bo, uint32(offset))
    
    // Entries
    for _, info := range this.Info {
        buf.Write(info.Marshal)
    }
    
    // Data result
    this.Data = buf.Bytes()
}

func (this *BaseClass) Load(mwref map[string][]WordRank) {
    // Insert words
    blist := map[string]*BaseInfo{}
    for key, wlist := range mwref {
        // Reprocess word list
        wnew := []WordRank{}
        for _, xlist := range wlist {
            // Existing word if any
            var wfound *WordRank
            wfound = nil
            
            // Loop old list looking for dupes
            for _, xnew := range wnew {
                if xnew.Info == xlist.Info {
                    if wfound == nil { wfound = &xnew }
                    if wfound.Rank > xlist.Rank { wfound.Rank = xlist.Rank }
                }
            }
            
            // Add if dupes not found
            if wfound == nil { wnew = append(wnew, xlist) }
        }
        
        // Entry
        blist[key] = &BaseInfo{
            Name: key,
            Wref: wnew,
        }
    }
    
    // Copy list
    for _, binfo := range blist {
        this.Info = append(this.Info, binfo)
    }
}

func (this *BaseClass) Save(fn string) {
    fs := DataOpen(fn)
    if (fs == nil) { return }
    fs.Write(this.Data)
    fs.Close()
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

func rune_decode(str string) []rune {
    arr := []rune{}
    for (len(str) > 0) {
        r, sz := utf8.DecodeRuneInString(str)
        if (sz == 0) { break }
        arr = append(arr, r)
        str = str[sz:]
    }
    return arr
}

// <===> Main <================================================================>
// Globals
var g_category *CategoryClass
var g_word *WordClass
var g_sentence *SentenceClass

// Main function
func main() {
    // Byte order
    g_bo = binary.LittleEndian
    
    // Classes
    g_category = CategoryNew()
    g_word = WordNew()
    g_sentence = SentenceNew()

    // Load
    fmt.Print("Loading categories...\n")
    if (!g_category.Load()) {
        fmt.Printf("Error parsing categories!\n")
        return
    }
    fmt.Print("Loading words...\n")
    if (!g_word.Load("words.xml")) {
        fmt.Printf("Error parsing words file!\n")
        return
    }
    fmt.Print("Loading sentences...\n")
    if (!g_sentence.Load("sentences.pipe")) {
        fmt.Printf("Error reading sentences file!\n")
        return
    }
    
    // Sentence search
    fmt.Print("Sentence search...\n")
    g_word.Search()
    
    // Id generation
    fmt.Printf("Assigning ids...\n")
    g_category.AssignId()
    g_word.AssignId()
    g_sentence.AssignId()
    
    // Marshalling
    fmt.Printf("Marshalling...\n")
    g_category.Marshal()
    g_word.Marshal()
    g_sentence.Marshal()
    
    // Save
    fmt.Print("Writing data...\n")
    g_category.Save("kotoba-category.kdb")
    g_word.Save("kotoba-word.kdb")
    g_sentence.Save("kotoba-sentence.kdb")
    
    // Bases
    fmt.Print("Generating bases...\n")
    base_k := BaseNew()
    base_k.Load(g_word.BaseReal)
    base_f := BaseNew()
    base_f.Load(g_word.BaseKana)
    base_e := BaseNew()
    base_e.Load(g_word.BaseEn)
    
    base_k.AssignId()
    base_f.AssignId()
    base_e.AssignId()
    
    base_k.Marshal()
    base_f.Marshal()
    base_e.Marshal()
    
    base_k.Save("kotoba-base_k.kdb")
    base_f.Save("kotoba-base_f.kdb")
    base_e.Save("kotoba-base_e.kdb")
}

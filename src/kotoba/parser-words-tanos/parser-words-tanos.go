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
    "encoding/csv"
    "encoding/hex"
    "strings"
    "crypto/sha1"
    "unicode"
    "unicode/utf8"
    "strconv"
)

/*
type JCConv struct {
    Hira map[rune]bool
    Kata map[rune]bool
}
var jcconv JCConv
*/

/*
func (this *JCConv) Init() {
    // Hiragana
    this.Hira = map[rune]bool{}
    hira_txt := "が ぎ ぐ げ ご ざ じ ず ぜ ぞ だ ぢ づ で ど ば び ぶ べ ぼ ぱ ぴ ぷ ぺ ぽ " +
                "あ い う え お か き く け こ さ し す せ そ た ち つ て と " +
                "な に ぬ ね の は ひ ふ へ ほ ま み む め も や ゆ よ ら り る れ ろ " +
                "わ を ん ぁ ぃ ぅ ぇ ぉ ゃ ゅ ょ っ"
    hira_arr := strings.Split(hira_txt, " ")
    for _, item := range hira_arr {
        r, _ := utf8.DecodeRuneInString(item)
        this.Hira[r] = true
    }
    
    // Katakana
    this.Kata = map[rune]bool{}
    kata_txt := "ガ ギ グ ゲ ゴ ザ ジ ズ ゼ ゾ ダ ヂ ヅ デ ド バ ビ ブ ベ ボ パ ピ プ ペ ポ " +
                "ア イ ウ エ オ カ キ ク ケ コ サ シ ス セ ソ タ チ ツ テ ト " +
                "ナ ニ ヌ ネ ノ ハ ヒ フ ヘ ホ マ ミ ム メ モ ヤ ユ ヨ ラ リ ル レ ロ " +
                "ワ ヲ ン ァ ィ ゥ ェ ォ ャ ュ ョ ッ"
    kata_arr := strings.Split(kata_txt, " ")
    for _, item := range kata_arr {
        r, _ := utf8.DecodeRuneInString(item)
        this.Kata[r] = true
    }
}
*/

func JconvRune(str string) []rune {
    arr := []rune{}
    for (len(str) > 0) {
        r, sz := utf8.DecodeRuneInString(str)
        if (sz == 0) { break }
        arr = append(arr, r)
        str = str[sz:]
    }
    return arr
}

func JconvText(arr []rune) string {
    str := ""
    buf := make([]byte, 16)
    for _, item := range arr {
        n := utf8.EncodeRune(buf, item)
        if (n > 0) {
            str += string(buf[0:n])
        }
    }
    return str
}

func JconvCharset(str string) int {
    arr := JconvRune(str)
    
    // Hiragana test
    is_hiragana := true
    for _, r := range arr {
        if (!unicode.Is(unicode.Hiragana, r)) {
            is_hiragana = false
            break
        }
    }
    if (is_hiragana) { return 1 }
    
    // Katakana test
    is_katakana := true
    for _, r := range arr {
        if (!unicode.Is(unicode.Katakana, r) && r != 'ー') {
            is_katakana = false
            break
        }
    }
    if (is_katakana) { return 2 }
    
    // Full cjk range
    rt := unicode.RangeTable{
        R16: []unicode.Range16{
            { Lo: 0x3000, Hi: 0x303f, Stride: 1 }, // Punctuation
            { Lo: 0x3040, Hi: 0x309f, Stride: 1 }, // Hiragana
            { Lo: 0x30a0, Hi: 0x30ff, Stride: 1 }, // Katakana
            { Lo: 0x3400, Hi: 0x4dbf, Stride: 1 }, // CJK unified ext A
            { Lo: 0x4e00, Hi: 0x9faf, Stride: 1 }, // CJK unified
            { Lo: 0xff00, Hi: 0xffef, Stride: 1 }, // Romanji and hw-katakana
        },
        R32: []unicode.Range32{},
        LatinOffset: 0,
    }
    is_cjk := true
    for _, r := range arr {
        if (!unicode.Is(&rt, r)) {
            is_cjk = false
            break
        }
    }
    if (is_cjk) { return 3 }
    
    // Failed to detect charset
    return 0
}

type Word struct {
    Id string
    Hash string
    JpText string
    JpReal string
    JpKana string
    En string
    Flags string
    Level int
}

type Entry struct {
    Id string
    A string
    B string
}

func load_submap(fn string) map[string]*Word {
    // File
    fs, err := os.Open(fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return nil
    }
    
    // Line reader
    ret := map[string]*Word{}
    reader := bufio.NewReader(fs)
    err = nil
    for err == nil {
        // Read line
        var line string
        line, err = reader.ReadString('\n')
        if (err != nil) { break }
        
        // Split
        record := strings.Split(line, "\t")
        if (len(record) < 6) {
            fmt.Printf("Error: Line does not have enough columns! num=%d\n", len(record))
            continue
        }
        
        // Info
        hash := strings.TrimSpace(record[0])
        jp_real := strings.TrimSpace(record[1])
        jp_kana := strings.TrimSpace(record[2])
        jp_text := jp_real
        if (len(jp_kana) > 0) {
            jp_text += "{" + jp_kana + "}"
        }
        level, _ := strconv.Atoi(strings.TrimSpace(record[5]))
        //fmt.Printf("conv '%s' -> '%d'\n", record[5], level)
        if (len(jp_real) > 0) {
            word := &Word{
                // Info
                Id: "",
                Hash: hash_sha1(jp_text),
                JpText: jp_text,
                JpReal: jp_real,
                JpKana: jp_kana,
                En: record[3],
                Flags: record[4],
                Level: level,
            }
            ret[hash] = word
        } else {
            ret[hash] = nil
        }
    }
    
    // Return
    return ret
}

func load(fn string) *[][]string {
    // File
    fs, err := os.Open(fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return nil
    }
    
    // CSV
    reader := csv.NewReader(fs)
    reader.LazyQuotes = true
    reader.FieldsPerRecord = -1
    var data [][]string
    data, err = reader.ReadAll()
    if (err != nil) {
        fmt.Printf("CSV parser error: " + err.Error() + "\n")
        return nil
    }
    return &data
}

func resample(raw *[][]string) []Entry {
    data := []Entry{}
    for _, arr := range *raw {
        a := unspan(arr[7])
        b := unspan(arr[8])
        if (a != "" && b != "" && b != "#NAME?") {
            data = append(data, Entry{
                Id: hash_sha1(a),
                A: a,
                B: b,
            })
        }
    }
    return data
}

func unspan(str string) string {
    str = strings.Trim(strings.TrimSpace(str), "\"")
    if (strings.HasPrefix(str, "<")) {
        idx := strings.Index(str, ">")
        if (idx > 0) { str = str[idx+1:] }
    }
    idx := strings.Index(str, "<")
    if (idx > 0) { str = str[0:idx] }
    return str
}

func hash_sha1(str string) string {
    sha := sha1.New()
    io.WriteString(sha, str)
    return hex.EncodeToString(sha.Sum(nil))
}

func proc(level int, fn_k string, fn_h string) []*Word {
    // Load xml files
    raw_k := load(fn_k)
    if (raw_k == nil) {
        fmt.Printf("Error parsing kanji file!\n")
        return nil
    }
    raw_h := load(fn_h)
    if (raw_h == nil) {
        fmt.Printf("Error parsing hiragana file!\n")
        return nil
    }
    
    // Resample data
    data_k := resample(raw_k)
    data_h := resample(raw_h)
    
    // Match
    ret := []*Word{}
    for _, entry := range data_k {
        // Kanji
        jp_real := entry.A
        if (len(jp_real) == 0) { continue }
        
        // English
        en_text := entry.B
        if (len(en_text) == 0) { continue }
        
        // Match hiragana
        jp_kana := ""
        for _, test := range data_h {
            if (test.A == entry.A) {
                jp_kana = test.B
                break
            }
        }
        
        // Flags
        flags := ""
        
        // Fix kanji
        jp_real = strings.Replace(jp_real, "\t", " ", -1)
        jp_real = strings.Replace(jp_real, "・", ";", -1)
        jp_real = strings.Replace(jp_real, "/", ";", -1)
        jp_real = strings.Replace(jp_real, " ", "", -1)
        jp_real = strings.Replace(jp_real, ";する", "", -1)
        switch JconvCharset(jp_real) {
            case 1: flags += "h"
            case 2: flags += "k"
        }
        
        // Fix hiragana
        if (strings.Index(jp_kana, "U") >= 0) { jp_kana = "" }
        jp_kana = strings.Replace(jp_kana, "\t", " ", -1)
        jp_kana = strings.Replace(jp_kana, "・", ";", -1)
        jp_kana = strings.Replace(jp_kana, "/", ";", -1)
        jp_kana = strings.Replace(jp_kana, " ", "", -1)
        idx_o := strings.Index(jp_kana, "（")
        idx_c := strings.Index(jp_kana, "）")
        if (idx_o >= 0 && idx_c > idx_o) {
            jp_kana = jp_kana[0:idx_o] + jp_kana[idx_c + 1:]
        }
        jp_kana = strings.Replace(jp_kana, "(", "", -1)
        jp_kana = strings.Replace(jp_kana, ")", "", -1)
        jp_kana = strings.Replace(jp_kana, ";する", "", -1)
        if (jp_kana == jp_real) { jp_kana = "" }
        
        jp_kana_split := strings.Split(jp_kana, ";")
        jp_kana = ""
        for _, item := range jp_kana_split {
            if (item == "する") { continue }
            
            if (JconvCharset(item) == 0) {
                fmt.Printf("Parser: Hiragana charser error! real='%s', kana='%s'\n", jp_real, item)
                continue
            }
            
            if (jp_kana != "") { jp_kana += ";" }
            jp_kana += item
        }
        
        // Fix english
        en_text = strings.Replace(en_text, "\t", " ", -1)
        en_split := strings.Split(en_text, ";")
        en_text = ""
        for _, item := range en_split {
            item = strings.TrimSpace(item)
            item = strings.Replace(item, "1.", "", -1)
            item = strings.Replace(item, "2.", "", -1)
            item = strings.Replace(item, "3.", "", -1)
            item = strings.Replace(item, "4.", "", -1)
            item = strings.Replace(item, "(1)", "", -1)
            item = strings.Replace(item, "(2)", "", -1)
            item = strings.Replace(item, "(3)", "", -1)
            item = strings.Replace(item, "(4)", "", -1)
            item = strings.Replace(item, "  ", " ", -1)
            item = strings.Replace(item, "  ", " ", -1)
            item = strings.Replace(item, "  ", " ", -1)
            item = strings.TrimSpace(item)
            if (en_text != "") { en_text += ";" }
            en_text += item
        }
        
        // Construct text
        jp_text := jp_real
        if (len(jp_kana) != 0) {
            jp_text += "{" + jp_kana + "}"
        }
        
        // Word
        word := &Word{
            Id: hash_sha1(jp_real + "|" + jp_kana + "|" + en_text + "|" + strconv.Itoa(level)),
            Hash: hash_sha1(jp_text),
            JpText: jp_text,
            JpReal: jp_real,
            JpKana: jp_kana,
            En: en_text,
            Flags: flags,
            Level: level,
        }
        
        // Array
        ret = append(ret, word)
    }
    
    // Return
    return ret
}

func smix(a []string, b []string) string {
    // Check
    if (len(a) == 1 && a[0] == "") { a = []string{} }

    // Mix arrays
    for _, x := range b {
        found := false
        for _, y := range a {
            if x == y { found = true }
        }
        if (!found && x != "") { a = append(a, x) }
    }
    
    // Serialize
    r := ""
    for _, x := range a {
        if (r != "") { r += ";" }
        r += x
    }
    return r
}

type CollisionDb struct {
    fs *os.File
    rw *bufio.ReadWriter
    ListSplit map[string]bool
    ListMerge map[string]*Word
}

func (this *CollisionDb) Open(fn string) bool {
    // Open
    var err error
    this.fs, err = os.OpenFile(fn, os.O_RDWR | os.O_CREATE, 0644)
    if err != nil {
        fmt.Printf("Failed to open collision db!\n")
        return false
    }
    
    // Initialize
    this.ListSplit = map[string]bool{}
    this.ListMerge = map[string]*Word{}
    
    // Read
    this.rw = bufio.NewReadWriter(bufio.NewReader(this.fs), bufio.NewWriter(this.fs))
    err = nil
    for err == nil {
        var line string
        line, err = this.rw.ReadString('\n')
        if (err != nil) { break }
        line = strings.TrimSpace(line)
        split := strings.Split(line, "\t")
        if (len(split) >= 1) {
            if (split[0] == "MERGE") {
                jp_text := split[2]
                if (len(split[3]) != 0) {
                    jp_text += "{" + split[3] + "}"
                }
                level, _ := strconv.Atoi(split[6])
                word := &Word{
                    Id: hash_sha1(split[2] + "|" + split[3] + "|" + split[4] + "|" + strconv.Itoa(level)),
                    Hash: hash_sha1(jp_text),
                    JpReal: split[2],
                    JpKana: split[3],
                    En: split[4],
                    Flags: split[5],
                    Level: level,
                }
                for err == nil {
                    line, err = this.rw.ReadString('\n')
                    if (err != nil) { break }
                    line = strings.TrimSpace(line)
                    if (len(line) > 5) {
                        this.ListMerge[line] = word
                    } else {
                        break
                    }
                }
            } else if (split[0] == "SPLIT") {
                for err == nil {
                    line, err = this.rw.ReadString('\n')
                    if (err != nil) { break }
                    line = strings.TrimSpace(line)
                    if (len(line) > 5) {
                        this.ListSplit[line] = true
                    } else {
                        break
                    }
                }
            }
        }
    }
    
    // Success
    return true
}

func (this *CollisionDb) WriteSplit(list []*Word) {
    str := "SPLIT\n"
    for _, word := range list {
        str += word.Id + "\n"
    }
    str += "\n"
    this.rw.WriteString(str)
    this.rw.Flush()
}

func (this *CollisionDb) WriteMerge(list []*Word, item *Word) {
    str := fmt.Sprintf("MERGE\t%s\t%s\t%s\t%s\t%s\t%d\n", item.Hash, item.JpReal, item.JpKana, item.En, item.Flags, item.Level)
    for _, word := range list {
        str += word.Id + "\n"
    }
    str += "\n"
    this.rw.WriteString(str)
    this.rw.Flush()
}

func (this *CollisionDb) ArrayMerge(slist []*Word, clist [][]*Word) []*Word {
    rlist := []*Word{}
    for _, sitem := range slist { rlist = append(rlist, sitem) }
    for _, sitem := range slist {
        for j := range clist {
            found := false
            for _, citem := range clist[j] {
                if citem == sitem { found = true }
            }
            if found {
                tlist := clist[j]
                clist[j] = []*Word{}
                for _, titem := range this.ArrayMerge(tlist, clist[0:]) {
                    rlist = append(rlist, titem)
                }
            }
        }
    }
    return rlist
}

func (this *CollisionDb) Execute(rmap map[string][]*Word) []*Word {
    clist := [][]*Word{}
    for _, rlist := range rmap {
        clist = append(clist, rlist)
    }
    
    for i := range clist {
        tlist := clist[i]
        clist[i] = []*Word{}
        clist[i] = snodupes(this.ArrayMerge(tlist, clist[0:]))
    }
    
    xlist := []*Word{}
    i := 0
    for _, citem := range clist {
        if (len(citem) > 1) {
            tlist := this.Resolve(citem, float32(i) / float32(len(clist)))
            for _, titem := range tlist {
                xlist = append(xlist, titem)
            }
        } else if (len(citem) == 1) {
            xlist = append(xlist, citem[0])
        }
        i += 1
    }
    
    return xlist;
}

func (this *CollisionDb) Resolve(list []*Word, perc float32) []*Word {
    // Collision details
    fmt.Printf("\n<===> Collision (%.01f%%) <============================================>\n", perc * 100.0)
    
    // Return list
    ret := []*Word{}
    
    // Predefined splits
    tlist := []*Word{}
    mmap := map[*Word][]string{}
    for _, item := range list {
        _, s_exists := this.ListSplit[item.Id]
        _, m_exists := this.ListMerge[item.Id]
        if (s_exists) {
            ret = append(ret, item)
            fmt.Printf("< SPLIT: [%s]\n", item.Id)
        } else if (m_exists) {
            word := this.ListMerge[item.Id]
            _, exists := mmap[word]
            if (!exists) {
                mmap[word] = []string{ item.Id }
            } else {
                mmap[word] = append(mmap[word], item.Id)
            }
        } else {
            tlist = append(tlist, item)
        }
    }
    for item, ids := range mmap {
        ret = append(ret, item)
        str := ""
        for _, id := range ids {
            str += "[" + id + "] "
        }
        fmt.Printf("< MERGE: %s\n", str)
    }
    list = tlist
    
    // Loop while still pending words
    for len(list) > 0 {
        // Print list
        for i, item := range list {
            fmt.Printf("[%d] id='%s', jp_k='%s', jp_h='%s', en='%s', level='%d'\n", i + 1, item.Id, item.JpReal, item.JpKana, item.En, item.Level)
        }
        
        // Get command
        fmt.Printf("> ")
        var cmd string
        _, err := fmt.Scanln(&cmd)
        
        // Get entry numbers
        cmd = strings.TrimSpace(cmd)
        im_num := 0
        im := map[int]bool{}
        if err == nil {
            for _, r := range []rune(cmd) {
                i, cerr := strconv.Atoi(string(r))
                if cerr == nil && i > 0 && i <= len(list) {
                    im[i - 1] = true
                    im_num += 1
                }
            }
        }
        
        if im_num == 0 {
        
            // Split all
            this.WriteSplit(list)
            for _, item := range list {
                ret = append(ret, item)
            }
            list = []*Word{}
            
        } else {
            
            // Split into two lists
            tlist = []*Word{}
            mlist := []*Word{}
            for i, item := range list {
                _, exists := im[i]
                if exists {
                    mlist = append(mlist, item)
                } else {
                    tlist = append(tlist, item)
                }
            }
            list = tlist
            
            // Check
            if len(mlist) <= 1 { continue }
            
            // Merge entry
            m_jp_real := ""
            m_jp_kana := ""
            m_en_text := ""
            m_flags := ""
            m_flags_h := true
            m_flags_k := true
            m_level := 1
            
            for _, item := range mlist {
                m_jp_real = sinsert(m_jp_real, item.JpReal)
                m_jp_kana = sinsert(m_jp_kana, item.JpKana)
                m_en_text = sinsert(m_en_text, item.En)
                m_flags_h = m_flags_h && strings.Contains(item.Flags, "h")
                m_flags_k = m_flags_k && strings.Contains(item.Flags, "k")
                if item.Level > m_level { m_level = item.Level }
            }
            
            if m_flags_h { m_flags += "h" }
            if m_flags_k { m_flags += "k" }
            
            m_jp_text := m_jp_real
            if (len(m_jp_kana) != 0) {
                m_jp_text += "{" + m_jp_kana + "}"
            }
            
            mword := &Word{
                Id: hash_sha1(m_jp_real + "|" + m_jp_kana + "|" + m_en_text + "|" + strconv.Itoa(m_level)),
                Hash: hash_sha1(m_jp_text),
                JpReal: m_jp_real,
                JpKana: m_jp_kana,
                En: m_en_text,
                Flags: m_flags,
                Level: m_level,
            }
            
            // Write
            this.WriteMerge(mlist, mword)
        }
    }
    
    // Return
    return ret;
}

func snodupes(list []*Word) []*Word {
    nlist := []*Word{}
    for _, item := range list {
        found := false
        for _, sitem := range nlist {
            if sitem == item { found = true }
        }
        if !found { nlist = append(nlist, item) }
    }
    return nlist
}

func sappend(a []*Word, b *Word) []*Word {
    for _, item := range a {
        if item == b { return a }
    }
    return append(a, b)
}

func sinsert(lstr string, sitem string) string {
    llist := strings.Split(lstr, ";")
    found := false
    for _, litem := range llist {
        if strings.TrimSpace(litem) == strings.TrimSpace(sitem) { found = true }
    }
    if !found {
        if len(lstr) > 0 {
            return lstr + ";" + sitem
        } else {
            return sitem
        }
    } else {
        return lstr
    }
}

func soverlap(a string, b string) bool {
    for _, ai := range strings.Split(a, ";") {
        for _, bi := range strings.Split(b, ";") {
            if ai == bi { return true }
        }
    }
    return false
}

// Main
func main() {
    // Get all words
    list_cat := [][]*Word {
        proc(5, "n5-real.csv", "n5-kana.csv"),
        proc(4, "n4-real.csv", "n4-kana.csv"),
        proc(3, "n3-real.csv", "n3-kana.csv"),
        proc(2, "n2-real.csv", "n2-kana.csv"),
        proc(1, "n1-real.csv", "n1-kana.csv"),
    }
    list := []*Word{}
    for _, slist := range list_cat {
        for _, sitem := range slist {
            list = append(list, sitem);
        }
    }
    
    // Kanji collision
    cdb_kanji := CollisionDb{}
    if (!cdb_kanji.Open("collision-kanji.db")) { return }
    rmap := map[string][]*Word{}
    for _, witem := range list {
        for _, str := range strings.Split(witem.JpReal, ";") {
            str = strings.TrimSpace(str)
            _, efound := rmap[str]
            if (!efound) {
                rmap[str] = []*Word{ witem }
            } else {
                rmap[str] = append(rmap[str], witem)
            }
        }
    }
    list = cdb_kanji.Execute(rmap)
    
    // Kana collision
    cdb_kana := CollisionDb{}
    if (!cdb_kana.Open("collision-kana.db")) { return }
    rmap = map[string][]*Word{}
    for _, witem := range list {
        lookup := witem.JpReal
        if (len(witem.JpKana) > 0) { lookup = witem.JpKana }
        for _, str := range strings.Split(lookup, ";") {
            str = strings.TrimSpace(str)
            _, efound := rmap[str]
            if (!efound) {
                rmap[str] = []*Word{ witem }
            } else {
                rmap[str] = append(rmap[str], witem)
            }
        }
    }
    list = cdb_kana.Execute(rmap)
    
    /*
    // English collision
    cdb_english := CollisionDb{}
    if (!cdb_english.Open("collision-english.db")) { return }
    rmap = map[string][]*Word{}
    for _, witem := range list {
        for _, str := range strings.Split(witem.En, ";") {
            str = strings.TrimSpace(str)
            _, efound := rmap[str]
            if (!efound) {
                rmap[str] = []*Word{ witem }
            } else {
                rmap[str] = append(rmap[str], witem)
            }
        }
    }
    list = cdb_english.Execute(rmap)
    */
    
    // Substitution filter
    submap := load_submap("words-rule2.csv");
    tlist := []*Word{}
    for _, item := range list {
        sword, sexists := submap[item.Hash]
        if sexists {
            if sword != nil { tlist = append(tlist, sword) }
        } else {
            tlist = append(tlist, item)
        }
    }
    list = tlist
    
    // Check for duplicates
    dupabort := false
    dupmap := map[string]*Word{}
    for _, item := range list {
        _, dexists := dupmap[item.Hash]
        if dexists {
            fmt.Printf("DUPLICATE: id='%s', jp_k='%s', jp_h='%s', en='%s'\n", item.Hash, item.JpReal, item.JpKana, item.En)
            dupabort = true
        } else {
            dupmap[item.Hash] = item
        }
    }
    if dupabort {
        fmt.Printf("Aborting due to duplicates!\n")
        return
    }
    
    // Output
    fs, err := os.OpenFile("words-tanos.pipe", os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if (err != nil) {
        fmt.Printf("Failed to open output file: %s\n", err.Error())
        return
    }
    for _, item := range list {
        fmt.Fprintf(fs, "%s\t%s\t%s\t%s\t%s\t%d\n", item.Hash, item.JpReal, item.JpKana, item.En, item.Flags, item.Level)
    }
}

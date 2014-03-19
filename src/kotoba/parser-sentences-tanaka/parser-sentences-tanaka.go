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
    "os/exec"
    "bytes"
    "bufio"
    "strings"
    "strconv"
    "unicode"
    "unicode/utf8"
)

type JCConv struct {
    kata2hira map[rune]rune
    hira map[rune]bool
    kata map[rune]bool
}
var jcconv JCConv

func (this *JCConv) Init() {
    // Hiragana
    hira_txt := "が ぎ ぐ げ ご ざ じ ず ぜ ぞ だ ぢ づ で ど ば び ぶ べ ぼ ぱ ぴ ぷ ぺ ぽ " +
                "あ い う え お か き く け こ さ し す せ そ た ち つ て と " +
                "な に ぬ ね の は ひ ふ へ ほ ま み む め も や ゆ よ ら り る れ ろ " +
                "わ を ん ぁ ぃ ぅ ぇ ぉ ゃ ゅ ょ っ"
    hira_arr := strings.Split(hira_txt, " ")
    this.hira = map[rune]bool{}
    for _, ch := range hira_arr {
        r, _ := utf8.DecodeRuneInString(ch)
        this.hira[r] = true
    }
    
    // Katakana
    kata_txt := "ガ ギ グ ゲ ゴ ザ ジ ズ ゼ ゾ ダ ヂ ヅ デ ド バ ビ ブ ベ ボ パ ピ プ ペ ポ " +
                "ア イ ウ エ オ カ キ ク ケ コ サ シ ス セ ソ タ チ ツ テ ト " +
                "ナ ニ ヌ ネ ノ ハ ヒ フ ヘ ホ マ ミ ム メ モ ヤ ユ ヨ ラ リ ル レ ロ " +
                "ワ ヲ ン ァ ィ ゥ ェ ォ ャ ュ ョ ッ"
    kata_arr := strings.Split(kata_txt, " ")
    this.kata = map[rune]bool{}
    for _, ch := range kata_arr {
        r, _ := utf8.DecodeRuneInString(ch)
        this.kata[r] = true
    }
    
    // Conversion map
    this.kata2hira = map[rune]rune{}
    for i, kata := range kata_arr {
        kata_r, _ := utf8.DecodeRuneInString(kata)
        hira_r, _ := utf8.DecodeRuneInString(hira_arr[i])
        this.kata2hira[kata_r] = hira_r
    }
}

func (this *JCConv) Rune(str string) []rune {
    arr := []rune{}
    for (len(str) > 0) {
        r, sz := utf8.DecodeRuneInString(str)
        if (sz == 0) { break }
        arr = append(arr, r)
        str = str[sz:]
    }
    return arr
}

func (this *JCConv) Text(arr []rune) string {
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

func (this *JCConv) ConvKataHira(str string) string {
    arr := this.Rune(str)
    for i, r := range arr {
        key, valid := this.kata2hira[r]
        if (valid) { arr[i] = key }
    }
    return this.Text(arr)
}

func (this *JCConv) IsHiragana(str_r []rune) bool {
    for _, r := range str_r {
        _, exists := this.hira[r]
        if !exists { return false }
    }
    return true
}

func (this *JCConv) Inject(text string, furi string) string {
    // Debug
    //fmt.Printf("* text='%s', furi='%s'\n", text, furi)
    
    // Runeify
    text_r := jcconv.Rune(text)
    furi_r := jcconv.Rune(furi)
    
    // Loop until out of text
    ret_s := ""
    ret_e := ""
    for len(text_r) > 0 && len(furi_r) > 0 {
        // Hiragana and katakana (at start)
        tc := text_r[0]
        tc_kval, tc_kex := this.kata2hira[tc]
        if tc_kex { tc = tc_kval }
        if tc == furi_r[0] {
            ret_s += this.Text(text_r[0:1])
            text_r = text_r[1:]
            furi_r = furi_r[1:]
            continue
        }
        
        // Hiragana and katakana (at end)
        tc = text_r[len(text_r) - 1]
        tc_kval, tc_kex = this.kata2hira[tc]
        if tc_kex { tc = tc_kval }
        if tc == furi_r[len(furi_r) - 1] {
            ret_e = this.Text(text_r[len(text_r) - 1:]) + ret_e
            text_r = text_r[0:len(text_r) - 1]
            furi_r = furi_r[0:len(furi_r) - 1]
            continue
        }
        
        // Kanji block
        text_sz := 0
        for text_sz < len(text_r) {
            _, exists := this.hira[text_r[text_sz]]
            if exists { break }
            _, exists = this.kata[text_r[text_sz]]
            if exists { break }
            text_sz += 1
        }
        if text_sz == 0 || len(furi_r) == 0 { panic(nil) }
        
        // Furigana block
        furi_sz := len(furi_r)
        if text_sz < len(text_r) && text_sz < furi_sz {
            furi_sz = text_sz
            for furi_sz < len(furi_r) {
                if furi_r[furi_sz] == text_r[text_sz] { break }
                furi_sz += 1
            }
        }
        
        // Add annoted kanji
        ret_s += "{" + this.Text(text_r[0:text_sz]) + ";" + this.Text(furi_r[0:furi_sz]) + "}"
        text_r = text_r[text_sz:]
        furi_r = furi_r[furi_sz:]
    }
    
    // Leftovers
    if len(text_r) > 0 { ret_s += this.Text(text_r) }
    
    // Success
    return ret_s + ret_e
}

type Word struct {
    Text string
    Kana string
    Base string
}

var SentenceDb map[string]bool

func load(fn string) {
    // File
    fs, err := os.Open(fn)
    if (err != nil) {
        fmt.Printf("Failed to open file: " + err.Error() + "\n")
        return
    }
    
    // Line reader
    fmt.Printf("Loading: ")
    reader := bufio.NewReader(fs)
    err = nil
    for err == nil {
        var line_a string
        line_a, err = reader.ReadString('\n')
        if (err != nil) { break }
        _, err = reader.ReadString('\n')
        if (err != nil) { break }
        parse(line_a)
        
        out_id += 1
        if out_id % 1000 == 0 { fmt.Printf("%dk ", out_id / 1000) }
    }
}

func parse(str string) {
    // Trim usless parts and split
    str = strings.TrimPrefix(str, "A: ")
    
    jp_ident := ""
    idx := strings.Index(str, "#ID")
    if (idx > 0) {
        jp_ident = str[idx + 3:]
        str = str[0:idx]
        jp_ident = strings.TrimSpace(jp_ident)
        jp_ident = strings.Trim(jp_ident, "=")
        idx = strings.Index(jp_ident, "_")
        if idx > 0 { jp_ident = jp_ident[idx + 1:] }
    }
    
    split := strings.SplitN(str, "\t", 2)
    if (len(split) != 2) {
        fmt.Printf("Error: Could not split sentence to jp and en parts!\n")
        return
    }
    
    jp_text := strings.Replace(strings.TrimSpace(split[0]), "\t", " ", -1)
    jp_text = strings.Replace(jp_text, ";", " ", -1)
    jp_text = strings.Replace(jp_text, "{", "[", -1)
    jp_text = strings.Replace(jp_text, "}", "]", -1)
    jp_text = strings.Replace(jp_text, "@", "(at)", -1)
    en_text := strings.Replace(strings.TrimSpace(split[1]), "\t", " ", -1)
    
    // Analysis
    jp_parse, jp_base := mecab(jp_text)
    if (jp_parse == "") { return }
    
    // Check dupes
    _, dup_found := SentenceDb[jp_parse]
    if (dup_found) { return }
    SentenceDb[jp_parse] = true
    
    // Output
    out_wr.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", jp_ident, jp_text, jp_parse, en_text, jp_base))
}

func mecab(str string) (string, string) {
    // Input sanitization
    str = strings.Replace(str, "|", "", -1)
    str = strings.Replace(str, "~", "", -1)
    
    // Execute
    cmd := exec.Command("mecab", "--node-format=\"%ps|%pe|%m|%f[7]|%f[6]~\"", "--eos-format=\"\n\"", "--unk-format=\"%ps|%pe|%m|%m|%m~\"")
    cmd.Stdin = strings.NewReader(str + "\n")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if (err != nil) {
        fmt.Printf("Error: Mecab failure: %s\n", err.Error())
        return "", ""
    }
    txt_full := out.String()
    
    // Result parsing
    txt_full = strings.TrimSpace(txt_full)
    txt_full = strings.Trim(txt_full, "~")
    txt_split := strings.Split(txt_full, "~")
    
    // Parse each word
    sentence := ""
    baselist := ""
    mark_off := 0
    for _, item := range txt_split {
        // Split
        split := strings.Split(item, "|")
        if (len(split) < 5) { continue }
        
        // Item
        word := Word{
            Text: split[2],
            Kana: split[3],
            Base: split[4],
        }
        
        // Check
        if (len(word.Text) == 0) { continue }
        
        // Marked text length
        mark_len := len(jcconv.Rune(word.Text))
        
        // Conditionals
        if (len(word.Kana) > 0) {
            arr := jcconv.Rune(word.Text)
            if (unicode.IsDigit(arr[0]) || unicode.IsPunct(arr[0])) {
                word.Kana = ""
                word.Base = ""
            }
        }
        if (word.Text == word.Kana) { word.Kana = "" }
        if (len(word.Kana) > 0) { word.Kana = jcconv.ConvKataHira(word.Kana) }
        if (word.Text == word.Kana) { word.Kana = "" }
        
        // Furigana insertion
        if (len(word.Kana) > 0) {
            word.Text = jcconv.Inject(word.Text, word.Kana)
        }
        
        // Sentence
        sentence += word.Text
        if (len(word.Base) > 0) {
            base_r := jcconv.Rune(word.Base)
            is_hiragana := jcconv.IsHiragana(base_r)
            if (!is_hiragana || len(base_r) > 1) {
                if (len(baselist) > 0) { baselist += ";" }
                base_h := word.Base
                if !is_hiragana { base_h = mecab_base(word.Base) }
                baselist += word.Base + "@" + base_h + "@" + strconv.Itoa(mark_off) + "@" + strconv.Itoa(mark_off + mark_len)
            }
        }
        mark_off += mark_len
    }
    
    // Debug
    //fmt.Printf("* %s  (%v)\n", sentence, baselist)
    
    // Return
    return sentence, baselist
}

func mecab_base(str string) string {
    // Input sanitization
    str = strings.Replace(str, "|", "", -1)
    str = strings.Replace(str, "~", "", -1)

    // Execute
    cmd := exec.Command("mecab", "--node-format=\"%ps|%pe|%m|%f[7]|%f[6]~\"", "--eos-format=\"\n\"", "--unk-format=\"%ps|%pe|%m|%m|%m~\"")
    cmd.Stdin = strings.NewReader(str + "\n")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if (err != nil) {
        fmt.Printf("Error: Mecab failure: %s\n", err.Error())
        return ""
    }
    txt_full := out.String()
    
    // Result parsing
    txt_full = strings.TrimSpace(txt_full)
    txt_full = strings.Trim(txt_full, "~")
    txt_split := strings.Split(txt_full, "~")
    
    // Split
    split := strings.Split(txt_split[0], "|")
    if (len(split) < 5) { return "" }
    
    // Return hiragana
    return jcconv.ConvKataHira(split[3]);
}

// File stream
var out_fs *os.File
var out_wr *bufio.Writer
var out_id int

// Main
func main() {
    // Flags
    fn_t := flag.String("tanaka", "", "Tanaka corpus file")
    flag.Parse()
    
    // Check
    if (*fn_t == "") {
        fmt.Printf("Please specify Tanaka corpus file!\n")
        return
    }
    
    // Output file
    out_id = 0
    var err error
    out_fs, err = os.OpenFile("sentences.pipe", os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if (err != nil) {
        fmt.Printf("Failed to open output file: %s\n", err.Error())
        return
    }
    out_wr = bufio.NewWriter(out_fs)
    
    // Charconv
    jcconv.Init()

    // Sentence db
    SentenceDb = map[string]bool{}

    // Load
    load(*fn_t)
}

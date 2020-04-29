package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"

	"github.com/tealeg/xlsx"
)

type textData struct{}
type wordData struct {
	ID        string
	text      string
	cleanText string
	lemma     string
	tag       string
}
type verse struct {
	kind    string
	number  int
	wordIDs []string
}

type index = map[string][]int

type vocabulary map[string][]string
type scholie map[string][]string

// Export structs
type verseExportData struct {
	Title  string          `json:"title"`
	N      int             `json:"n"`
	Verses [][]interface{} `json:"verses"`
}

// TODO alignment structures

func main() {
	var dataFolder = flag.String("data", "input-data", "Data folder with xslx version. Each filename will be used as text id.")
	flag.Parse()

	fileNames, err := getFilePaths(*dataFolder)
	if err != nil {
		log.Fatalln(err)
	}

	// textInfo := []textData{}
	for _, file := range fileNames {
		_, _, err := parseExcel(file)
		// info, err := getTextInfo(file)
		if err != nil {
			log.Fatalln(err)
		}
		// textInfo = append(textInfo, info)
	}
}

func mergeMaps(x, y map[string]wordData) map[string]wordData {
	data := map[string]wordData{}
	for k, v := range x {
		data[k] = v
	}
	for k, v := range y {
		data[k] = v
	}
	return data
}

func getWords(path string) map[string]wordData {
	// id = text + id + id2
	// read data from excel
	return map[string]wordData{} // TODO
}

func getVerses(path string) []verse {
	return []verse{} // TODO
}

func getTextInfo(path string) (textData, error) {
	return textData{}, nil // TODO
}

func getIndex(words map[string]wordData) index {
	return index{} // TODO
}

func getFilePaths(dirPath string) ([]string, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames := []string{}
	for _, f := range files {
		fileNames = append(fileNames, fmt.Sprint(dirPath, "/", f.Name()))
	}

	return fileNames, nil
}

func getWord(row *xlsx.Row) wordData {
	return wordData{
		ID:        fmt.Sprint(row.Cells[2].Value, "-", row.Cells[0].Value, "-", row.Cells[1].Value),
		text:      row.Cells[15].Value,
		cleanText: row.Cells[19].Value,
		lemma:     row.Cells[20].Value,
		tag:       row.Cells[21].Value,
	}
}

func getVerseInfo(row *xlsx.Row) (int, string, int, error) { // get book, kind, verse number or error
	book, err := strconv.Atoi(row.Cells[3].Value)
	if err != nil {
		return 0, "", 0, err
	}
	kind := getVerseKind(row)
	var verseNumber int
	if kind == "f" {
		verseNumber = math.MaxInt32
	}
	if kind != "f" && kind != "t" {
		verseNumber, err = strconv.Atoi(row.Cells[11].Value)
		if err != nil {
			return 0, "", 0, err
		}
	}
	if err != nil {
		return 0, "", 0, err
	}
	return book, kind, verseNumber, nil
}

func parseExcel(path string) (map[string]wordData, map[int][]verse, error) { // returns all words and verses divided by book
	xlFile, err := xlsx.OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	words := map[string]wordData{} // wordID -> word
	versesByBook := map[int][]verse{}

	lastBook := -1
	lastKind := ""
	lastVerseNum := -1

	for _, sheet := range xlFile.Sheets {
		for i, row := range sheet.Rows {
			if i == 0 || row.Cells[0].Value == "" {
				continue
			}

			word := getWord(row)
			words[word.ID] = word

			book, kind, verseNumber, err := getVerseInfo(row)
			if err != nil {
				return nil, nil, err
			}
			// nuovo verso quando, cambia il libro oppure cambia il tipo oppure cambia il numero di verso
			if book != lastBook || lastKind != kind || lastVerseNum != verseNumber {
				// setup new Verse!
				if _, exists := versesByBook[book]; !exists {
					versesByBook[book] = []verse{}
				}
				versesByBook[book] = append(versesByBook[book], verse{kind: kind, number: verseNumber, wordIDs: []string{}})
			}
			// Append word ID
			versesByBook[book][len(versesByBook[book])-1].wordIDs = append(versesByBook[book][len(versesByBook[book])-1].wordIDs, word.ID)

			lastBook = book
			lastKind = kind
			lastVerseNum = verseNumber
			// se nuovo
			// versesByBook[book] = nil // nuovo verso con il tipo giusto
			// altimenti appendi
			// versesByBook[book][len(versesByBook)-1].wordIDs = append(versesByBook[book][len(versesByBook)-1].wordIDs, word.ID)

		}
	}

	return words, versesByBook, nil
}

// for _, e := range entries {
// 	if bookVersesMap[e.book] == nil {
// 		bookVersesMap[e.book] = map[int]verseEntry{}
// 	}

// 	if bookVersesMap[e.book][e.verse].kind == "" {
// 		bookVersesMap[e.book][e.verse] = verseEntry{
// 			kind:   e.kind,
// 			verse:  e.verse,
// 			verses: []string{},
// 		}
// 	}

// 	bookVersesMap[e.book][e.verse] = verseEntry{
// 		kind:   bookVersesMap[e.book][e.verse].kind,
// 		verse:  bookVersesMap[e.book][e.verse].verse,
// 		verses: append(bookVersesMap[e.book][e.verse].verses, e.word),
// 	}
// }

func getVerseKind(row *xlsx.Row) string {
	switch row.Cells[4].Value {
	case "Tit.":
		return "t"
	case "Omisit":
		return "o"
	case "Des.":
		return "f"
	default:
		return "v"
	}
}

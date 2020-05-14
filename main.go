package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/szenzaro/iliad-aligner/aligner"
	"github.com/tealeg/xlsx"
)

func main() {
	dataFolder := flag.String("data", "input-data", "Data folder with xslx version. Each filename will be used as text id.")
	vocPath := flag.String("voc", "data/Vocabulaire_Genavensis.xlsx", "path to the vocabulary xlsx file")
	scholiePath := flag.String("sch", "data/scholied.json", "path to the scholie JSON file")
	flag.Parse()

	fileNames, err := getFileNames(*dataFolder)
	if err != nil {
		log.Fatalln(err)
	}

	if err := os.RemoveAll("out"); err != nil {
		log.Fatalln(err)
	}
	start := time.Now()
	texts := map[string]textInfo{}
	for _, file := range fileNames {
		fmt.Println()

		words, verses, err := parseExcel(fmt.Sprint(*dataFolder, "/", file))
		if err != nil {
			log.Fatalln(err)
		}

		textName := getTextName(file)
		texts[textName] = textInfo{words, verses}

		folder := "texts/" + textName + "/"

		fmt.Println("Generate text info")
		generateTextData(folder, words, verses)

		fmt.Println("Generate Search indexes")
		generateIndex(words, folder+"/index/", "Lemma")
		generateIndex(words, folder+"/index/", "Text")

		fmt.Println()
	}
	generateAlignments(fileNames, texts, *vocPath, *scholiePath)
	elapsed := time.Since(start)
	fmt.Println("Generation time needed: ", elapsed)
}

type wordData struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Chant     string `json:"chant"`
	Verse     string `json:"verse"`
	CleanText string `json:"normalized"`
	Lemma     string `json:"lemma"`
	Tag       string `json:"tag"`
	Source    string `json:"source"`
}
type verse struct {
	Kind    string
	Number  int
	WordIDs []string
}

type index = map[string][]int

type vocabulary map[string][]string
type scholie map[string][]string

type textInfo struct {
	words  map[string]wordData
	verses map[int][]verse
}

func getTextName(fileName string) string { return fileName[:len(fileName)-5] }

func computeAlignments(tasks map[string]aligner.Problem, ar aligner.Aligner, ff []aligner.Feature, w []float64, subseqLen int) map[string]*aligner.Alignment {

	resAlignments := map[string]*aligner.Alignment{}

	start := time.Now()
	for k, p := range tasks {
		fmt.Println("Aligning ", k)
		a := aligner.NewFromWordBags(p.From, p.To)
		res, err := a.Align(ar, ff, w, subseqLen, aligner.AdditionalData)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println()
		fmt.Println("Got: ", res)
	}
	elapsed := time.Since(start)
	fmt.Println("Alignment time needed: ", elapsed)
	return resAlignments
}

func getProblems(source, target string, sourceWords, targetWords map[string]wordData) map[string]aligner.Problem {
	data := map[string]aligner.Problem{}
	for _, w := range sourceWords {
		problemID := fmt.Sprintf("%s.%s", w.Chant, w.Verse)
		if _, ok := data[problemID]; !ok {
			if problemID == "" {
				panic("AA") // TODO
			}
			data[problemID] = aligner.Problem{From: map[string]aligner.Word{}, To: map[string]aligner.Word{}}
		}
		data[problemID].From[w.ID] = getAlignerWord(w)
	}
	for _, w := range targetWords {
		problemID := fmt.Sprintf("%s.%s", w.Chant, w.Verse)
		if _, ok := data[problemID]; !ok {
			if problemID == "" {
				panic("AA") // TODO
			}
			data[problemID] = aligner.Problem{From: map[string]aligner.Word{}, To: map[string]aligner.Word{}}
		}
		data[problemID].To[w.ID] = getAlignerWord(w)
	}
	return data
}

func getAlignerWord(w wordData) aligner.Word {
	return aligner.Word{ID: w.ID, Text: w.Text, Lemma: w.Lemma, Tag: w.Tag, Verse: w.Verse, Chant: w.Chant, Source: w.Source}
}

func generateAlignment(sourceText, targetText string, textInfo map[string]textInfo, ff []aligner.Feature, w []float64, subseqLen int) (map[string]*aligner.Alignment, error) {
	fmt.Println("Generating alignment for ", sourceText, " - ", targetText)

	problems := getProblems(sourceText, targetText, textInfo[sourceText].words, textInfo[targetText].words)
	ar := aligner.NewGreekAligner()

	aligments := map[string]*aligner.Alignment{}

	for id, p := range problems {
		start := time.Now()
		fmt.Println("Align in progres for ", id)
		a := aligner.NewFromWordBags(p.From, p.To)
		res, err := a.Align(ar, ff, w, subseqLen, aligner.AdditionalData)
		if err != nil {
			return nil, fmt.Errorf("generateAlignment %s: %w", id, err)
		}
		aligments[id] = res
		elapsed := time.Since(start)
		fmt.Println(id, " has been aligned in ", elapsed)
	}
	return aligments, nil
}

func generateAlignments(fileNames []string, textInfo map[string]textInfo, vocPath, scholiePath string) error {
	w := []float64{0.2956361042981355, 0.060325626401096885, 0.033855873309357465, 0.024419617049442562, 0.8058173377380647, 0.004187020307669374, 0.1931506936628718}

	ff := []aligner.Feature{
		aligner.EditType,
		aligner.LexicalSimilarity,
		aligner.LemmaDistance,
		aligner.TagDistance,
		aligner.VocDistance,
		aligner.ScholieDistance,
		aligner.MaxDistance,
	}

	subseqLen := 5
	aligner.AdditionalData = map[string]interface{}{}
	fmt.Println("Loading vocabulary")
	_, err := aligner.LoadVoc(vocPath)
	if err != nil {
		return err
	}

	fmt.Println("Loading scholie")
	_, err = aligner.LoadScholie(scholiePath)
	if err != nil {
		return err
	}
	for i, sourceFile := range fileNames {
		for j := i + 1; j < len(fileNames); j++ {
			targetFile := fileNames[j]
			source, target := getTextName(sourceFile), getTextName(targetFile)
			alignments, err := generateAlignment(source, target, textInfo, ff, w, subseqLen)
			if err != nil {
				return fmt.Errorf("generateAlignments: %w", err)
			}
			sourceJSONEdits, targetJSONEdits := alignments["asd"].ToJSONEdits()

			// Save to JSON
			dir := "out/alignments/auto/" + source + "/"
			writeToJSON(dir, dir+target+".json", sourceJSONEdits)
			dir = "out/alignments/auto/" + target + "/"
			writeToJSON(dir, dir+source+".json", targetJSONEdits)
		}
	}

	return nil
}

func generateIndex(words map[string]wordData, folder, fieldName string) {
	index := getIndexWords(words, fieldName)
	dir := "out/" + folder
	writeToJSON(dir, dir+strings.ToLower(fieldName)+".json", index)
}

func wordsByChant(words map[string]wordData) map[string]map[string]wordData {
	d := map[string]map[string]wordData{}
	for k, v := range words {
		if _, present := d[v.Chant]; !present {
			d[v.Chant] = map[string]wordData{}
		}
		d[v.Chant][k] = v
	}
	return d
}

func generateTextData(folder string, words map[string]wordData, verses map[int][]verse) {
	dir := "out/" + folder
	writeToJSON(dir, dir+"/words.json", wordsToExportData(words))

	for k, v := range wordsByChant(words) {
		newdir := dir + k
		writeToJSON(newdir, newdir+"/words.json", wordsToExportData(v))
	}

	for k, v := range versesToExportData(verses) {
		newdir := fmt.Sprintf("%s%v", dir, k)
		writeToJSON(newdir, newdir+"/verses.json", v)
	}
}

func writeToJSON(folder, path string, data interface{}) {
	fmt.Println("Savind JSON to ", path)

	if err := os.MkdirAll(folder, 0777); err != nil {
		log.Fatalln(err)
	}

	d, err := json.Marshal(data)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile(path, d, 0664)
	if err != nil {
		log.Fatalln(err)
	}
}

func wordsToExportData(ws map[string]wordData) map[string][]interface{} {
	d := map[string][]interface{}{}
	for k, w := range ws {
		d[k] = []interface{}{w.Text, w.CleanText, w.Lemma, w.Tag, w.Verse}
	}
	return d
}

func getIndexWords(words map[string]wordData, fieldName string) map[string][]string {
	uniqueWords := map[string][]string{}
	for _, v := range words {
		fieldValue := reflect.ValueOf(v).FieldByName(fieldName).String()
		addToMap(v.ID, fieldValue, &uniqueWords)
	}
	for _, v := range uniqueWords {
		sort.Strings(v)
	}
	return uniqueWords
}

func addToMap(id string, key string, data *map[string][]string) {
	if key == "" {
		return
	}
	_, present := (*data)[key]
	if !present {
		(*data)[key] = []string{}
	}
	(*data)[key] = append((*data)[key], id)
}

func versesToExportData(vs map[int][]verse) map[int][]interface{} {
	d := map[int][]interface{}{}
	for k, w := range vs {
		d[k] = []interface{}{}
		for i := range w {
			var v []interface{}
			switch vs[k][i].Kind {
			case "t":
				v = []interface{}{w[i].Kind, w[i].WordIDs}
			case "o":
				v = []interface{}{w[i].Kind, w[i].Number}
			case "f":
				v = []interface{}{w[i].Kind, w[i].WordIDs}
			default:
				v = []interface{}{w[i].Kind, w[i].Number, w[i].WordIDs}
			}
			d[k] = append(d[k], v)
		}
	}
	return d
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

func getFileNames(dirPath string) ([]string, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames := []string{}
	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}

	return fileNames, nil
}

func getWord(row *xlsx.Row) wordData {
	return wordData{
		ID:        fmt.Sprint(row.Cells[2].Value, "-", row.Cells[0].Value, "-", row.Cells[1].Value),
		Text:      row.Cells[15].Value,
		CleanText: row.Cells[19].Value,
		Lemma:     row.Cells[20].Value,
		Tag:       row.Cells[21].Value,
		Chant:     row.Cells[3].Value,
		Verse:     row.Cells[10].Value,
		Source:    row.Cells[2].Value,
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
	fmt.Println("Parsing ", path)
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
				versesByBook[book] = append(versesByBook[book], verse{Kind: kind, Number: verseNumber, WordIDs: []string{}})
			}
			// Append word ID
			versesByBook[book][len(versesByBook[book])-1].WordIDs = append(versesByBook[book][len(versesByBook[book])-1].WordIDs, word.ID)

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

# iliad-data-packer
Pack the data to be used in the iliad-viewer application


words: Map<{ text, lemma, ... }
verses: [ VerseType, number, data... ]

index: Map<int[]>

alignment: Map<{ type:string, target: string[]> // ID -> [ "type", ["id1", "id1", ...]]

ordered_alignment: Map<{ type:string, target: string[]>
id -> ids Map<{type:String, target: string[]}>

vocabulary: Map<string[]>
scholie: Map<string[]>


ExcelH, ExcelP, ExcelFR -> words, verses
words -> index


index ->

data
    manifest.json
    vocabulary.json
    - manuscript
        annotations.json
        books-to-pages.json
        manifest.json
        pages-to-verses.json
    - alignments
        - auto
            - text1
                text2.json
                text3.json
                ...
            - text1
                text1.json
                text3.json
                ...
        - manual
    + texts
        + text1
            - 1 (chant number)
                verses.json
                data.json
            - 2 
            - ...
        - text2
        - ...
        + index
            lemma.json
            text.json
    
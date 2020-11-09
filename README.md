# iliad-data-packer
This application packs the data to be used in the Iliadoscope (iliad-viewer) application.

This application is part of the SNF project [Le devenir numérique d'un texte fondateur : l'Iliade et le Genavensis Græcus 44](http://p3.snf.ch/Project-172733)

All the files in the data, data_backup, input-data, input-data copy directories are under the CC BY-NC-ND 4.0 license.
[![License: CC BY-NC-ND 4.0](https://img.shields.io/badge/License-CC%20BY--NC--ND%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-nd/4.0/)

The packer will produce a foldere structure like the one below
```
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
    - texts
        - text1
            - 1 (chant number)
                verses.json
                data.json
            - 2 
            - ...
        - text2
        - ...
        - index
            lemma.json
            text.json
```
